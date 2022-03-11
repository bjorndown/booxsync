import fetch from 'node-fetch'
import { readdir, readFile } from 'fs/promises'
import { join } from 'path'
import debug from 'debug'

const logHttp = debug('http')
const logDiff = debug('diff')

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

const walkLibrary = async (
  host: string,
  library: Library
): Promise<string[]> => {
  const allFiles = await Promise.all(
    library.visibleLibraryList.map(async (visibleLibrary: VisibleLibrary) => {
      if (!visibleLibrary.childCount) {
        return []
      }

      const files = []

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

      if (visibleLibrary.libraryCount > 0) {
        files.push(await walkLibrary(host, subLibrary))
      }

      const books = subLibrary.visibleBookList.map((book) =>
        join(visibleLibrary.name, book.name)
      )

      files.push(books)

      return files.flat()
    })
  )
  return allFiles.flat()
}

const fetchJson = async <T = Record<string, unknown>>(
  host: string,
  path: string
): Promise<T> => {
  logHttp(path)
  const response = await fetch(`${host}${path}`)
  return response.json()
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

const findFilesNotInLibrary = async ({
  host,
  syncRoot,
  skipPaths,
}: SyncConfig): Promise<string[]> => {
  const library: Library = await fetchJson(host, '/api/library')
  const filesInLibrary = new Set(await walkLibrary(host, library))

  const localFiles = await walkLocalFiles(syncRoot)

  return localFiles.filter((localFilePath) => {
    const fileRelativeToSyncRoot = localFilePath.replace(syncRoot, '')
    const notInLibrary = !filesInLibrary.has(fileRelativeToSyncRoot)
    const notToBeSkipped = !skipPaths.some((pathToSkip) =>
      fileRelativeToSyncRoot.includes(pathToSkip)
    )
    return notInLibrary && notToBeSkipped
  })
}

readFile('./config.json')
  .then((buffer) => JSON.parse(buffer.toString()))
  .then((config) => findFilesNotInLibrary(config))
  .then((filesNotInLibrary) => {
    console.log(filesNotInLibrary.length, 'files not in library')
    console.log(filesNotInLibrary.join('\n'))
  })
  .catch((error) => console.error(error))
