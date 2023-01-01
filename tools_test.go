package toolkit

// go test -coverprofile=coverage.out && go tool cover -html=coverage.out

import (
    "fmt"
    "image"
    "image/png"
    "io"
    "mime/multipart"
    "net/http/httptest"
    "os"
    "sync"
    "testing"
)

const randomStringLength = 10
const expectedRandomStringLength = 10

func TestTools_RandomString(t *testing.T) {
    var testTools Tools
    s := testTools.RandomString(randomStringLength)
    if len(s) != expectedRandomStringLength {
        t.Error("wrong length random string returned...")
    } else {
        t.Log("RandomString.length is equal to ", randomStringLength)
    }
}

var uploadTests = []struct{
    name string
    allowedTypes []string
    renameFile bool
    errorExpected bool
}{
    {
        name: "Allowed no rename",
        allowedTypes: []string{"image/jpeg", "image/png"},
        renameFile: false,
        errorExpected: false,
    },{
        name: "Allowed to rename file",
        allowedTypes: []string{"image/jpeg", "image/png"},
        renameFile: true,
        errorExpected: false,
    },{
        name: "Not allowed no rename",
        allowedTypes: []string{"image/jpeg"},
        renameFile: false,
        errorExpected: true,
    },
}

func TestTools_UploadFiles(t *testing.T) {
    for _, e := range uploadTests {
        // setup a pipe to avoid buffering
        pipeReader, pipeWriter := io.Pipe()
        writer := multipart.NewWriter(pipeWriter)

        // Adding a wait group for things to in sync and nothing is breaking
        waitGroup := sync.WaitGroup{}
        waitGroup.Add(1)

        // fire a go routine
        go func() {
            // Close when the file upload is done and finished
            defer writer.Close()

            // Decrement the wait group and close
            defer waitGroup.Done()

            // create a form data payload to send data
            multiPartFile, err := writer.CreateFormFile("file", "./testdata/img.png")
            if err != nil {
                t.Error(err)
            }

            f, err := os.Open("./testdata/img.png")
            if err != nil {
                t.Error(err)
            }
            defer f.Close()

            // Decode the image
            img, _, err := image.Decode(f)
            if err != nil {
                t.Error("error decoding image", err)
            }

            err = png.Encode(multiPartFile, img)
            if err != nil {
                t.Error(err)
            }
        }()

        // read from the pipe which received data
        // set request headers and url
        request := httptest.NewRequest("POST", "/", pipeReader)
        request.Header.Add("Content-Type", writer.FormDataContentType())

        var testTools Tools
        testTools.AllowedFileTypes = e.allowedTypes

        uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
        if err != nil && !e.errorExpected {
            t.Error(err)
        }

        if !e.errorExpected {
            // if file is not present from the test data
            if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].FileName)); os.IsNotExist(err) {
                t.Errorf("%s expected file to exists %s", e.name, err.Error())
            }

            // clean up and then delete the file which we have just moved/copied
             _ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].FileName))
        }

        if err != nil && !e.errorExpected {
            t.Errorf("%s: error expected but none received", e.name)
        }

        waitGroup.Wait()
    }
}

func TestTools_UploadOneFile(t *testing.T) {
    // setup a pipe to avoid buffering
    pipeReader, pipeWriter := io.Pipe()
    writer := multipart.NewWriter(pipeWriter)

    // Adding a wait group for things to in sync and nothing is breaking
    // waitGroup := sync.WaitGroup{}
    // waitGroup.Add(1)

    // fire a go routine
    go func() {
        // Close when the file upload is done and finished
        defer writer.Close()

        // Decrement the wait group and close
        // defer waitGroup.Done()

        // create a form data payload to send data
        multiPartFile, err := writer.CreateFormFile("file", "./testdata/img.png")
        if err != nil {
            t.Error(err)
        }

        f, err := os.Open("./testdata/img.png")
        if err != nil {
            t.Error(err)
        }
        defer f.Close()

        // Decode the image
        img, _, err := image.Decode(f)
        if err != nil {
            t.Error("error decoding image", err)
        }

        err = png.Encode(multiPartFile, img)
        if err != nil {
            t.Error(err)
        }
    }()

    // read from the pipe which received data
    // set request headers and url
    request := httptest.NewRequest("POST", "/", pipeReader)
    request.Header.Add("Content-Type", writer.FormDataContentType())

    var testTools Tools
    // testTools.AllowedFileTypes = e.allowedTypes

    uploadedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
    if err != nil {
        t.Error(err)
    }

    // if file is not present from the test data
    if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.FileName)); os.IsNotExist(err) {
        t.Errorf("expected file to exists %s", err.Error())
    }

    // clean up and then delete the file which we have just moved/copied
    _ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.FileName))

}

func TestTools_CreateDirIfNotExists(t *testing.T) {
    var testTools Tools

    err := testTools.CreateDirIfNotExists("./testdata/myDir")
    if err != nil {
        t.Error(err)
    }

    err = testTools.CreateDirIfNotExists("./testdata/myDir")
    if err != nil {
        t.Error(err)
    }

    _ = os.RemoveAll("./testdata/myDir")
}

var slugTests = []struct {
    name string
    s string
    expected string
    errorExpected bool
} {
    {name: "valid string", s: "now is the time", expected: "now-is-the-time", errorExpected: false},
    {name: "empty string", s: "", expected: "", errorExpected: true},
    {name: "complex string", s: "Now is the time for all GOOD men! + fish & such &^123", expected: "now-is-the-time-for-all-good-men-fish-such-123", errorExpected: false},
    {name: "japanese string", s: "こんにちは世界", expected: "", errorExpected: true},
    {name: "japanese string and roman characters", s: "hello world こんにちは世界", expected: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
    var testTools Tools

    for _, e := range slugTests {
        slug, err := testTools.Slugify(e.s)
        if err != nil && !e.errorExpected {
            t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
        }

        if !e.errorExpected && slug != e.expected {
            t.Errorf("%s: wrong slug returned; expected %s got: %s", e.name, e.expected, slug)
        }
    }
}