package file

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseYAML(t *testing.T) {
	filename := "/tmp/yaml-test-parse"
	testYAML := []byte("foo: bar\nmoo: zoo")
	err := ioutil.WriteFile(filename, testYAML, 0644)
	defer os.Remove(filename)
	assert.NoError(t, err)

	procArgs := make(map[string]string)
	err = ParseYAML(filename, procArgs)
	assert.NoError(t, err)
	assert.Equal(t, procArgs["foo"], "bar")
	assert.Equal(t, procArgs["moo"], "zoo")
}

func TestParseYAMLError(t *testing.T) {

	errorTests := []struct {
		Filename     string
		ErrorMessage string
	}{
		{"/tmp/foo", "no such file or directory"},
		{"/tmp/yaml-test-parse-error", "cannot unmarshal"},
	}

	filename := "/tmp/yaml-test-parse-error"
	testYAML := []byte("foo bar")
	err := ioutil.WriteFile(filename, testYAML, 0644)
	defer os.Remove(filename)
	assert.NoError(t, err)

	for _, errorTest := range errorTests {
		procArgs := make(map[string]string)
		err = ParseYAML(errorTest.Filename, procArgs)
		assert.Contains(t, err.Error(), errorTest.ErrorMessage)
	}
}
