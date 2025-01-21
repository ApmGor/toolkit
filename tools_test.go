package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools
	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("wrong length random string returned")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// set up pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := new(sync.WaitGroup)
		wg.Add(1)
		go func() {
			defer func(writer *multipart.Writer) {
				_ = writer.Close()
			}(writer)
			defer wg.Done()
			/// Create the form data field "file"
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			file, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer func(file *os.File) {
				_ = file.Close()
			}(file)
			img, _, err := image.Decode(file)
			if err != nil {
				t.Error("error decoding image", err)
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// read from the pipe which receives data
		request := httptest.NewRequest(http.MethodPost, "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())
		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes
		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}
		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}
			// clean up
			t.Cleanup(func() {
				for {
					err := os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
					if err == nil {
						break
					}
				}
			})
		}
		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", e.name)
		}
		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// set up pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer func(writer *multipart.Writer) {
			_ = writer.Close()
		}(writer)
		/// Create the form data field "file"
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		file, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer func(file *os.File) {
			_ = file.Close()
		}(file)
		img, _, err := image.Decode(file)
		if err != nil {
			t.Error("error decoding image", err)
		}
		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	// read from the pipe which receives data
	request := httptest.NewRequest(http.MethodPost, "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())
	var testTools Tools
	uploadedFile, err := testTools.UploadOneFile(request, "./testdata/uploads/")
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}
	// clean up
	t.Cleanup(func() {
		for {
			err := os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName))
			if err == nil {
				break
			}
		}
	})
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTool Tools
	err := testTool.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	err = testTool.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll("./testdata/myDir")
	})
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "now is the time", expected: "now-is-the-time", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{
		name:          "complex string",
		s:             "  Now is the time for all GOOD men! + fish & such &^123    <",
		expected:      "now-is-the-time-for-all-good-men-fish-such-123",
		errorExpected: false,
	},
	{name: "japanese string", s: "こんにちは世界", expected: "", errorExpected: true},
	{name: "japanese and roman string", s: "hello world こんにちは世界", expected: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTool Tools
	for _, e := range slugTests {
		slug, err := testTool.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
		}
		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: wrong slug returned; expected %s but got %s", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	var testTool Tools
	testTool.DownloadStaticFile(w, r, "./testdata", "pic.jpg", "puppy.jpg")
	res := w.Result()
	if res.Header["Content-Length"][0] != "98827" {
		t.Errorf("wrong Content-Length of %s", res.Header["Content-Length"][0])
	}
	if res.Header["Content-Disposition"][0] != "attachment; filename=\"puppy.jpg\"" {
		t.Error("wrong Content-Disposition")
	}
	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
	t.Cleanup(func() {
		_ = res.Body.Close()
	})
}
