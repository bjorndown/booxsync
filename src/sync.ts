import debug from 'debug'
import { readdir, readFile } from 'fs/promises'
import fetch, { fileFrom, FormData } from 'node-fetch'
import { basename, join } from 'path'

const logHttp = debug('boox-sync:http')
const logDiff = debug('boox-sync:diff')

type Library = {
  bookCount: number
  libraryCount: number
  visibleLibraryList: VisibleLibrary[]
  visibleBookList: VisibleBook[]
}

type VisibleLibrary = {
  name: string
  title: string
  idString: string
  childCount: number // number of books
  libraryCount: number // number of libraries
}

type VisibleBook = {
  name: string
  metadata: {
    _id: string
  }
}

type LibraryApiParams = {
  limit?: number
  offset?: number
  sortBy?: string
  order?: 'Desc' | 'Asc'
  libraryUniqueId: string
}

type SyncConfig = {
  host: string
  syncRoot: string
  skipPaths: string[]
}

const walkLibraries = async (
  host: string,
  visibleLibraries: VisibleLibrary[],
  root = '/'
): Promise<{ files: string[]; pathToId: Record<string, string> }> => {
  const walkLibrary = async (
    host: string,
    root: string,
    visibleLibrary: VisibleLibrary
  ): Promise<{ files: string[]; id: string; path: string }[]> => {
    if (!visibleLibrary.childCount) {
      return []
    }

    const currentPath = join(root, visibleLibrary.name)

    const params: LibraryApiParams = {
      libraryUniqueId: visibleLibrary.idString,
    }

    const queryParams = new URLSearchParams({
      args: JSON.stringify(params),
    })

    logDiff('visiting', visibleLibrary.title)

    const subLibrary: Library = await fetchJson(
      host,
      `/api/library?${queryParams}`
    )

    const subLibs = await Promise.all(
      subLibrary.visibleLibraryList.map((visibleLibrary) =>
        walkLibrary(host, currentPath, visibleLibrary)
      )
    )

    const books = subLibrary.visibleBookList.map((book) =>
      join(currentPath, book.name)
    )

    logDiff(`books in '${currentPath}':`, books)

    subLibs.push([
      { files: books, id: visibleLibrary.idString, path: currentPath },
    ])

    return subLibs.flat()
  }

  const allFiles = (
    await Promise.all(visibleLibraries.map((l) => walkLibrary(host, root, l)))
  ).flat()

  const pathToId = Object.fromEntries(allFiles.map((l) => [l.path, l.id]))

  return { files: allFiles.map((l) => l.files).flat(), pathToId }
}

const fetchJson = async <T = Record<string, unknown>>(
  host: string,
  path: string
): Promise<T> => {
  logHttp(path)
  const response = await fetch(`${host}${path}`)
  return response.json() as unknown as T // TODO
}

const walkLocalFiles = async (path: string): Promise<string[]> => {
  const files = await readdir(path, { withFileTypes: true })

  const filesInTree = await Promise.all(
    files.map(async (file) => {
      if (file.isDirectory()) {
        return walkLocalFiles(join(path, file.name))
      }
      return join(path, file.name)
    })
  )

  return filesInTree.flat()
}

const upload = async (
  config: SyncConfig,
  path: string,
  parentFolder: string
) => {
  const formData = new FormData()
  const filename = basename(path)

  formData.set('sender', 'web')
  formData.set('file', await fileFrom(path, 'application/pdf'), filename)
  formData.set('parent', parentFolder)

  const response = await fetch(`${config.host}/api/library/upload`, {
    method: 'POST',
    body: formData as any,
  })

  if (response.ok) {
    return response.json()
  }

  const responseText = await response.text()
  throw new Error(
    `upload of ${path} failed with ${response.status} ${response.statusText}, body: ${responseText}`
  )
}

const findFilesNotInLibrary = async ({
  host,
  syncRoot,
  skipPaths,
}: SyncConfig): Promise<{ file: string; parentId: string | undefined }[]> => {
  const library: Library = await fetchJson(host, '/api/library')
  const { files, pathToId } = await walkLibraries(
    host,
    library.visibleLibraryList
  )
  const filesInLibrary = new Set(files)

  const localFiles = await walkLocalFiles(syncRoot)

  return localFiles
    .filter((localFilePath) => {
      const fileRelativeToSyncRoot = localFilePath.replace(syncRoot, '')
      const notInLibrary = !filesInLibrary.has(fileRelativeToSyncRoot)
      const notToBeSkipped = !skipPaths.some((pathToSkip) =>
        fileRelativeToSyncRoot.includes(pathToSkip)
      )
      return notInLibrary && notToBeSkipped
    })
    .map((localFilePath) => {
      const fileRelativeToSyncRoot = localFilePath.replace(syncRoot, '')
      const parentId = pathToId[fileRelativeToSyncRoot]
      return { file: localFilePath, parentId }
    })
}

const main = async () => {
  try {
    const config = JSON.parse((await readFile('./config.json')).toString())
    const filesNotInLibrary = await findFilesNotInLibrary(config)

    console.log(`${filesNotInLibrary.length} files missing in Boox library`)

    for (const { file, parentId } of filesNotInLibrary) {
      if (parentId) {
        console.log(`uploading ${file} to ${parentId}`)
        await upload(config, file, parentId)
      } else {
        console.warn(`skipping '${file}' because parent does exist`)
      }
    }

    console.log('Boox library synced')
  } catch (error: any) {
    if (error.errno === 'EHOSTUNREACH') {
      console.log(`Boox device unreachable. Is it asleep? If not, try opening the BooxDrop app in the browser`)
    } else {
      console.error(error)
    }
  }
}

main()
