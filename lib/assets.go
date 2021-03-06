// Code generated by go-bindata.
// sources:
// assets/VERSION
// assets/agent-commands.html
// DO NOT EDIT!

package insight_server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _assetsVersion = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x2a\x33\xd4\x33\xd6\x33\xe6\x02\x04\x00\x00\xff\xff\x89\xcf\xf0\x67\x07\x00\x00\x00")

func assetsVersionBytes() ([]byte, error) {
	return bindataRead(
		_assetsVersion,
		"assets/VERSION",
	)
}

func assetsVersion() (*asset, error) {
	bytes, err := assetsVersionBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/VERSION", size: 7, mode: os.FileMode(420), modTime: time.Unix(1464073245, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _assetsAgentCommandsHtml = []byte("\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xb4\x56\x6d\x53\xdb\xba\x12\xfe\xce\xaf\xd8\xfa\x76\x86\x84\x62\x3b\x21\xa5\xa5\x34\xe1\x0e\x85\xdc\x5e\x18\xde\xda\x50\xe8\xcb\xf4\x83\xb0\x37\x8e\x82\x2c\x19\x49\x4e\x48\x3b\xfd\xef\x77\x65\x3b\x89\x43\xe9\x1d\xce\x99\x39\x99\x69\xf1\x4a\xbb\xcf\xee\x3e\xab\x5d\xa9\xfb\xec\xf0\xfc\xe0\xf2\xcb\x45\x1f\x46\x36\x15\x7b\x6b\x5d\xf7\x07\x04\x93\x49\xcf\x43\xe9\xb9\x05\x64\xf1\xde\x1a\xd0\xaf\x9b\xa2\x65\x10\x8d\x98\x36\x68\x7b\x5e\x6e\x87\xfe\x8e\x57\xdf\x1a\x59\x9b\xf9\x78\x97\xf3\x49\xcf\xfb\xec\x7f\xda\xf7\x0f\x54\x9a\x31\xcb\x6f\x04\x7a\x10\x29\x69\x51\x92\xdd\x51\xbf\x87\x71\x82\x2b\x96\x92\xa5\xd8\xf3\x26\x1c\xa7\x99\xd2\xb6\xa6\x3c\xe5\xb1\x1d\xf5\x62\x9c\xf0\x08\xfd\x42\xd8\x04\x2e\xb9\xe5\x4c\xf8\x26\x62\x02\x7b\xed\x39\xd0\x33\xdf\x87\xcb\x11\x02\xbb\x51\x13\x84\x0e\x14\xc0\x96\x25\x06\x36\xd2\xdc\xd8\x0d\x02\x4d\x11\x86\x5c\x1b\x4b\x10\x60\x49\xd5\xe5\xf6\x16\x98\x9c\x81\x22\x51\x17\xf2\xdc\x37\x38\xa3\xd2\x66\x83\x0d\x2d\xea\x0d\x67\x62\xb0\x84\xf4\xfd\xca\xab\xe5\x56\xe0\xde\x05\x45\x62\x2d\xc2\x91\x34\x3c\x19\x59\xd8\x4f\x1c\x02\xa5\x9f\x32\x19\x9b\x6e\x58\x6a\xad\x2d\x03\x7d\xa7\x94\x35\x56\xb3\xac\x40\x5a\xae\x9f\x30\x8b\xa5\xdb\x8c\x0b\x8c\x29\xb6\x18\x52\x4a\x78\xc8\x49\x38\x18\x0c\x96\x8e\x05\x97\xb7\xa0\x51\xf4\x3c\x63\x67\x02\xcd\x08\x91\x88\x1b\x69\x1c\xf6\x3c\x57\x08\xb3\x1b\x86\x29\xbb\x8f\x62\x19\xdc\xcc\x9d\x39\x81\xa0\xc3\xc5\x42\xd8\x09\x3a\xc1\xab\x30\x32\x66\xb9\x16\x90\xbf\x80\x56\x3c\x62\xc9\x62\xa2\xb9\x9d\x91\x8f\x11\xeb\xec\xbc\xf4\xdb\x77\x3b\xe9\xe5\xf1\xf9\xfe\xe0\x7e\x67\xdc\xde\xcf\x5f\xb0\xed\xeb\xc3\x2b\x79\xc1\xb7\xc4\xed\x7f\x86\xd3\x69\x7f\x9f\xed\x8c\x0e\x0f\xe3\xf1\x57\x91\x9d\x60\x72\x3f\x1a\x5f\x9d\xf6\xdb\xc3\x64\x7c\x7d\xf1\x3e\xbd\xfd\x61\x5e\x53\x65\xb5\x32\x46\x69\x9e\x70\xd9\xf3\x98\x54\x72\x96\xaa\xdc\x78\x75\x0e\xce\x33\xcb\x95\x64\xc2\xf1\x4d\xec\xff\xf3\x19\xfb\x85\xa3\xff\x97\xf7\xf0\xe4\x7a\xeb\xac\xd5\x16\xa7\x77\x63\x76\xfb\xee\xf6\xbe\x23\xc2\xd3\x37\x7d\x36\xca\xa7\xd9\x60\x88\x67\x93\xab\x57\x9d\xe3\x6d\xfc\x21\x3b\xf9\xd7\x1f\x2c\xbb\x6c\xe5\xaf\xfb\x5f\xcc\xe7\xd3\xf1\x87\xab\x17\xad\xbe\xdc\xd6\x4f\xca\xfb\xbf\x97\xa7\x27\xdb\x60\x46\x3c\x2d\xca\xfe\x11\x4d\xa6\x64\x1c\x8c\x0d\x0c\x95\x86\xa3\xfe\x0e\x98\x3c\x73\xfd\x01\x6a\x58\x29\xa3\xa0\xc0\xa5\x35\xe5\x39\xc1\x98\x33\xb8\xcb\x51\x73\xac\x9d\x50\x07\x7d\xbd\xff\xf1\xec\xe8\xec\xfd\x6e\x1d\x34\x56\x68\xe4\xba\x85\xa9\xd2\xb7\xc0\x87\x30\x53\x39\xb8\x0e\x2c\x3a\x23\x63\x09\x92\xc4\xa8\x5f\x04\x12\xaf\x2b\x70\xdf\x48\x5b\x58\x8a\x08\xde\x7c\xaf\x56\x4d\xa4\x79\x66\xc1\xe8\x68\x59\x0b\x4a\x38\xa8\xea\xe1\x4a\xe0\x26\xcb\x36\x65\x37\xa1\x12\xbc\x0e\xb6\x96\x72\x41\xfc\x98\xb8\xe8\x86\x25\xcc\xd3\x31\x75\x99\x4e\xd8\x0e\x5e\x12\x62\x25\xfd\x09\xef\xd9\x37\x94\x31\x1f\x7e\x77\xa9\x74\xc3\x72\xae\x75\x6f\x54\x3c\xa3\x22\xac\x75\x63\x3e\x81\x48\x30\x63\x7a\x9e\x9b\x00\x8c\x4b\xd4\xf3\xd1\x52\xdb\xd3\x6a\x5a\xad\x3e\xdc\x89\x14\x8d\xa4\xd4\x6f\x6f\xd5\xf6\x1f\xea\x38\x5a\x7d\xe7\x79\x81\xbd\xa2\x39\x6a\xef\x1d\x90\x6f\xad\x04\x55\x54\x08\x60\x49\x51\x5c\xe5\xa6\x15\x37\x60\x50\x4f\x50\x53\xe8\xed\x07\x1e\x42\x72\x51\x9d\xa4\xbf\xe7\xf5\xb7\xc5\xb2\x00\x19\x93\x73\x0c\xfa\xdf\xfa\x51\x39\xd0\x0a\x66\x69\xef\x4f\x66\xa9\x8b\xbd\x6e\x67\xa9\x18\x27\x8a\xc5\x5c\x26\x60\x2c\xb3\xb9\x09\x82\x80\x30\x9c\xe2\x23\x01\x3d\x2d\x41\x56\xb5\xfe\xbf\x3c\x22\x28\x12\x3c\xba\xa5\x76\xa5\x0a\x57\x53\xb7\xb1\x4e\x9e\xb4\x5d\x6f\x7a\xf3\x50\x6e\xac\x04\xfa\xe7\x67\x9a\xa7\x4c\xcf\x8a\xef\x7b\xe1\xed\x0d\x9c\x5e\x37\x64\x7b\x7f\x19\x5f\x65\x4f\x82\x57\xd9\x0a\x7a\x95\x4d\xed\x73\x9e\x5f\xd1\xab\xe3\x0f\xd4\xc1\x33\x68\x48\x8c\xd0\x18\x87\xe4\xda\x7f\x71\x61\xac\x1b\x38\x66\x13\x36\x28\x9b\x23\x13\x39\x4d\x14\xd3\x2c\xfa\xf3\xd1\x8e\x61\x63\x76\x1f\x24\x4a\x25\x02\x59\xc6\x4d\xd1\x36\x6e\x2d\x14\xfc\xc6\x84\x63\x37\x2e\x66\xd4\x3f\xed\x76\xd0\xa9\xa4\x47\xfa\xa7\x08\xec\x88\x58\xc8\x63\x2c\x4e\xe6\xe2\x82\xaa\x02\x80\xc6\x0d\x0a\x35\x6d\x6e\x02\xc5\xca\x2b\x45\x4e\xed\x36\xe1\x71\x4e\xa3\xdc\x8d\x11\x3a\xd3\x06\x24\x62\x4c\x66\xbf\x85\x3b\x7e\x78\x03\xad\x06\x30\x57\x2e\x69\x9b\x30\x0d\xb5\x52\x40\x0f\x86\xb9\x8c\xdc\xad\xd1\xa8\x8e\x68\x13\x7e\x2e\xf8\x7e\x1e\xb8\x7c\x1b\x3f\x57\xca\x9b\x6b\xb1\x0b\xeb\x21\x51\x12\x4e\xda\x61\x65\xb5\xbe\xb9\xa2\x63\x67\x19\x92\xd2\xc5\xa7\xcb\x07\x1b\x31\xb3\x6c\x17\xbc\xca\xaa\xe7\xc1\x0b\xa8\xbe\x57\xf5\x4c\x1e\xb9\x1a\xee\x2e\xc3\x73\x96\xf5\xd8\xe6\x3f\xba\xd6\xa8\x3f\x06\x45\x6f\x34\x9a\x2b\xdb\xbf\x16\xd2\xaf\xe6\xdb\xb5\x72\x65\xc1\x42\xdd\xae\x4e\x43\xf3\xe7\x5a\x2d\xff\x04\xed\xf1\xe0\xfc\xac\x01\xde\x3c\x53\x43\xc3\x32\xa2\xe1\xf2\x6f\x7a\xea\x30\x7a\x68\x65\xe5\x03\xc6\xdb\x5c\x42\x14\x59\xc2\xc3\x60\x9f\x37\xbc\x60\x65\x16\x34\x03\x8b\xf7\xb6\x48\x2c\x98\x93\xff\xb8\x05\x4d\x81\xba\xb2\x35\xcd\x95\xcc\x16\x02\xdd\x36\x65\x5a\x80\x13\xd7\x09\x9d\x16\x15\x9b\x66\x72\x6c\xdc\x35\x35\x45\xc0\x85\x2a\x3d\x44\x2f\x79\x8a\x2a\xb7\x8d\x3a\x13\x9b\xf0\xaa\x05\x1b\xd0\x6e\xb5\x5a\xa5\x8f\x5f\x25\x71\xcf\x57\xb4\xc8\x67\xed\x84\x87\xe5\x4d\xd0\x0d\xcb\x87\xf0\xff\x02\x00\x00\xff\xff\x3d\xee\x4b\xe1\x19\x0b\x00\x00")

func assetsAgentCommandsHtmlBytes() ([]byte, error) {
	return bindataRead(
		_assetsAgentCommandsHtml,
		"assets/agent-commands.html",
	)
}

func assetsAgentCommandsHtml() (*asset, error) {
	bytes, err := assetsAgentCommandsHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "assets/agent-commands.html", size: 2841, mode: os.FileMode(420), modTime: time.Unix(1475576400, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"assets/VERSION":             assetsVersion,
	"assets/agent-commands.html": assetsAgentCommandsHtml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"assets": &bintree{nil, map[string]*bintree{
		"VERSION":             &bintree{assetsVersion, map[string]*bintree{}},
		"agent-commands.html": &bintree{assetsAgentCommandsHtml, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
