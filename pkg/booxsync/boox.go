package booxsync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	API_PATH = "/api/library"
)

type BooxFile struct {
	Name     string
	Id       string
	Children []*BooxFile
	IsDir    bool
	Parent   *BooxFile
}

type BooxLibrary struct {
	Root   *BooxFile
	Config *SyncConfig
}

type libraryJson struct {
	BookCount          int                  `json:"bookCount"`
	LibraryCount       int                  `json:"libraryCount"`
	VisibleLibraryList []visibleLibraryJson `json:"visibleLibraryList"`
	VisibleBookList    []visibleBookJson    `json:"visibleBookList"`
}

type visibleLibraryJson struct {
	IdString string `json:"idString"`
	Name     string `json:"name"`
}

type visibleBookJson struct {
	Name     string `json:"name"`
	Metadata struct {
		Id string `json:"_id"`
	} `json:"metadata"`
}

type createFolderResponseJson struct {
	Code       int  `json:"code"`
	Successful bool `json:"successful"`
	Data       struct {
		IdString string `json:"idString"`
		Name     string `json:"name"`
	} `json:"data"`
}
type createFolderRequestJson struct {
	Parent string `json:"parent,omitempty"`
	Name   string `json:"name"`
}

func (library *BooxLibrary) Stat(name string) (*BooxFile, error) {
	if name == "." {
		return library.Root, nil
	}

	tokens := strings.Split(filepath.Clean(name), string(filepath.Separator))

	currentPath := library.Root

	for i := 0; i < len(tokens); i++ {
		matched := false
		for _, child := range currentPath.Children {
			if tokens[i] == child.Name {
				currentPath = child
				matched = true
				break
			}
		}

		if !matched {
			return nil, fmt.Errorf("boox stat: %q in %q: %w", tokens[i], name, fs.ErrNotExist)
		}
	}

	return currentPath, nil
}

func (library *BooxLibrary) Exists(name string) (bool, error) {
	_, err := library.Stat(name)
	return err == nil, err
}

func (library *BooxLibrary) GetParentId(name string) (string, error) {
	file, err := library.Stat(name)
	if err == nil {
		return file.Parent.Id, nil
	}
	return "", err
}

func (library *BooxLibrary) CreateFolder(name string, parent *BooxFile) error {
	if library.Config.DryRun {
		log.Printf("pretending to create folder %q", path.Join(parent.Name, name))
		parent.Children = append(parent.Children, &BooxFile{Name: name, Id: "dryRun", IsDir: true})
		return nil
	}

	if !parent.IsDir {
		return fmt.Errorf("createFolder: parent must be a folder: %v", parent)
	}

	payload := createFolderRequestJson{Name: name, Parent: parent.Id}
	body, err := json.Marshal(payload)

	if err != nil {
		return fmt.Errorf("createFolder: marshal failed: %w", err)
	}

	response, err := http.Post(fmt.Sprintf("%s%s", library.Config.Host, API_PATH), "application/json", bytes.NewReader(body))

	if err != nil || response.StatusCode != 200 {
		return fmt.Errorf("createFolder: API call failed: %w", err)
	}

	defer response.Body.Close()
	body, err = io.ReadAll(response.Body)

	if err != nil {
		return fmt.Errorf("createFolder: reading body failed: %w", err)
	}

	var responseJson createFolderResponseJson
	err = json.Unmarshal(body, &responseJson)

	if err != nil {
		return fmt.Errorf("createFolder: unmarshalling body failed: %w", err)
	}

	parent.Children = append(parent.Children, &BooxFile{Name: responseJson.Data.Name, Id: responseJson.Data.IdString, IsDir: true})

	return nil
}

func (library *BooxLibrary) Upload(localPath string, parent *BooxFile) error {
	if library.Config.DryRun {
		log.Printf("pretending to upload file %q", localPath)
		return nil
	}

	body := &bytes.Buffer{}

	formWriter := multipart.NewWriter(body)

	err := formWriter.WriteField("sender", "web")
	if err != nil {
		return fmt.Errorf("upload: could not write field 'sender': %w", err)
	}

	err = formWriter.WriteField("parent", parent.Id)
	if err != nil {
		return fmt.Errorf("upload: could not write field 'parent': %w", err)
	}

	bodyWriter, err := formWriter.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return fmt.Errorf("upload: could not write field 'file': %w", err)
	}

	b, err := os.ReadFile(path.Join(library.Config.SyncRoot, localPath))

	if err != nil {
		return fmt.Errorf("upload: reading local file failed: %w", err)
	}

	_, err = bodyWriter.Write(b)

	if err != nil {
		return fmt.Errorf("upload: writing file to form failed: %w", err)
	}

	err = formWriter.Close()
	if err != nil {
		return fmt.Errorf("upload: closing form writer failed: %w", err)
	}

	response, err := http.Post(fmt.Sprintf("%s/api/library/upload", library.Config.Host), formWriter.FormDataContentType(), body)

	if err != nil || response.StatusCode != 200 {
		return fmt.Errorf("upload: http request failed: %w", err)
	}

	return nil
}

func (library *BooxLibrary) PrintFileTree(stopAt int) {
	printSubTree(library.Root, 0, stopAt)
	log.Println()
}

func printSubTree(file *BooxFile, level int, stopAt int) {
	if level == stopAt {
		return
	}

	for _, child := range file.Children {
		log.Println(strings.Repeat("\t", level), child.Name)
		printSubTree(child, level+1, stopAt)
	}
}

func walk(visibleLibrary visibleLibraryJson, config *SyncConfig) (*BooxFile, error) {
	query := url.Values{}
	// TODO fixme
	query.Add("args", fmt.Sprintf("{\"libraryUniqueId\":\"%s\"}", visibleLibrary.IdString))

	response, err := http.Get(fmt.Sprintf("%s%s?%s", config.Host, API_PATH, query.Encode()))

	if err != nil {
		return nil, fmt.Errorf("walk: http request failed: %w", err)
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)

	if err != nil {
		return nil, fmt.Errorf("walk: reading body failed: %w", err)
	}

	var library libraryJson
	err = json.Unmarshal(body, &library)

	if err != nil {
		return nil, fmt.Errorf("walk: unmarshalling body failed: %w", err)
	}

	folder := BooxFile{Name: visibleLibrary.Name, Id: visibleLibrary.IdString, IsDir: true}

	for _, book := range library.VisibleBookList {
		folder.Children = append(folder.Children, &BooxFile{Name: book.Name, Parent: &folder})
	}

	for _, subLibrary := range library.VisibleLibraryList {
		child, err := walk(subLibrary, config)
		if err != nil {
			return nil, fmt.Errorf("walk: error walking sub-library %q: %w", visibleLibrary.Name, err)
		}
		child.Parent = &folder
		folder.Children = append(folder.Children, child)
	}

	return &folder, nil
}

func GetBooxLibrary(config *SyncConfig) (*BooxLibrary, error) {
	response, err := http.Get(fmt.Sprintf("%s%s", config.Host, API_PATH))
	if err != nil {
		return nil, fmt.Errorf("getBooxLibrary: API call failed: %w", err)
	}

	var rootLibrary visibleLibraryJson
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("getBooxLibrary: failed to read body: %w", err)
	}

	err = json.Unmarshal(body, &rootLibrary)
	if err != nil {
		return nil, fmt.Errorf("getBooxLibrary: unmarshalling failed: %w", err)
	}

	root, err := walk(rootLibrary, config)
	if err != nil {
		return nil, fmt.Errorf("getBooxLibrary: walking library failed: %w", err)
	}

	return &BooxLibrary{Root: root, Config: config}, nil
}
