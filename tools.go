package toolkit

import (
    "crypto/rand"
    "errors"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path"
    "path/filepath"
    "regexp"
    "strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module. Any variable of this type
// will have access to all the methods with reciever *Tools
type Tools struct {
    MaxFileSize int
    AllowedFileTypes []string
}

// RandomString return a string of random characters of length n, using randomStringSource
// as source for the string
func (t *Tools) RandomString(n int) string {
    s, r := make([]rune, n), []rune(randomStringSource)
    for i := range s {
        p, _ := rand.Prime(rand.Reader, len(r))
        x, y := p.Uint64(), uint64(len(r))
        s[i] = r[x%y]
    }
    return string(s)
}

type UploadedFile struct {
    FileName string
    OriginalFileName string
    FileSize int64
}

// UploadOneFile is just a convenience method that calls UploadFiles, but expects only one file to
// be in the upload.
func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
    renameFile := true
    if len(rename) > 0 {
        renameFile = rename[0]
    }

    files, err := t.UploadFiles(r, uploadDir, renameFile)
    if err != nil {
        return nil, err
    }

    return files[0], nil
}

func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
    // Check if user wants to rename files or not
    // by default we shall always rename files
    renameFile := true
    if len(rename) > 0 {
        renameFile = rename[0]
    }

    var uploadedFiles []*UploadedFile

    // set MaxFileSize if we haven't set any values
    if t.MaxFileSize == 0 {
        t.MaxFileSize = 1024 * 1024 * 1024 // GB of filesize
    }

    // Create directory
    errDir := t.CreateDirIfNotExists(uploadDir)
    if errDir != nil {
        return nil, errDir
    }

    err := r.ParseMultipartForm(int64(t.MaxFileSize))
    if err != nil {
        errorPMF := "failed paring uploaded files, the uploaded file is to big"
        log.Println(errorPMF, err)
        return nil, errors.New(errorPMF)
    }

    for _, fHeaders := range r.MultipartForm.File {
        for _, hdr := range fHeaders {
            uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
                var uploadedFile UploadedFile

                infile, err := hdr.Open()
                if err != nil {
                    return nil, err
                }
                defer infile.Close()

                // read the file type
                buff := make([]byte, 512)

                // read file into buff, it gets first 512 bytes and checks for an error
                _, err = infile.Read(buff)
                if err != nil {
                    return nil, err
                }

                // TODO: Check to see if file type is permitted
                allowed := false
                fileType := http.DetectContentType(buff)
                //  allowedTypes := []string{
                //      "images/jpeg",
                //      "images/png",
                //      "images/gif",
                //  }

                // Check if the given file type is allowed and then set the flag appropriately
                if len(t.AllowedFileTypes) > 0 {
                    for _, x := range t.AllowedFileTypes {
                        if strings.EqualFold(fileType, x) {
                            allowed = true
                        }
                    }
                } else {
                    // All file types are allowed
                    allowed = true
                }

                // For some reason if files are not allowed to upload
                if !allowed {
                    return nil, errors.New("the uploaded file type is not permitted")
                }

                // Read the file from start and check for errors
                _, err = infile.Seek(0, 0)
                if err != nil {
                    return nil, err
                }

                // If we want to rename files else use the original file name
                if renameFile {
                    uploadedFile.FileName = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(hdr.Filename))
                } else {
                    uploadedFile.FileName = hdr.Filename
                }
                uploadedFile.OriginalFileName = hdr.Filename

                // Save file to disk
                var outFile *os.File
                defer outFile.Close()

                if outFile, err = os.Create(filepath.Join(uploadDir, uploadedFile.FileName)); err != nil {
                    return nil, err
                } else {
                    fileSize, err := io.Copy(outFile, infile)
                    if err != nil {
                        return nil, err
                    }
                    uploadedFile.FileSize = fileSize
                }

                uploadedFiles = append(uploadedFiles, &uploadedFile)
                return uploadedFiles, nil

            }(uploadedFiles)

            if err != nil {
                return uploadedFiles, err
            }
        }
    }
    return uploadedFiles, nil
}

// CreateDirIfNotExists creates a directory, and all necessary parents if it doesnt exists
func (t *Tools) CreateDirIfNotExists(path string) error {
    const mode = 0755
    if _, err := os.Stat(path); os.IsNotExist(err) {
        err := os.MkdirAll(path, mode)
        if err != nil {
            return err
        }
    }

    return nil
}

// Slugify is a very simple means of creating a slug from a string
func (t *Tools) Slugify(s string) (string, error) {
    // Check if we have a valid string to generate slug
    if s == "" {
        return "", errors.New("empty string not permitted")
    }

    // Slugify the string
    var regExp = regexp.MustCompile(`[^a-z\d]+`)
    slug := strings.Trim(regExp.ReplaceAllString(strings.ToLower(s), "-"), "-")

    // Check if we have atleast one single character after generating slug
    if len(slug) == 0 {
        return "", errors.New("after removing characters, slug is or zero length")
    }

    // return the generated slugh with an error of nil
    return slug, nil
}

// DownloadStaticFile downloads a file, and tries to force the browser to avoid displaying it
// in the browser window by setting content disposition. It alos allows specification of the
// display name
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, pathName, fileName, displayName string)  {
    filePath := path.Join(pathName, fileName)
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=\"%s\"", displayName))
    http.ServeFile(w, r, filePath)
}