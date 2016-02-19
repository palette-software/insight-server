package controllers

import (
	"github.com/palette-software/insight-server/app/models"
	"github.com/revel/revel"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

// The application controller itself
type Stats struct {
	*revel.Controller
}

type StatsOutput struct {
	Directories []DirStatEntry
}

type DirStatEntry struct {
	Name   string
	Size   int64
	Tenant string
}

// Calculates the size of a directory
func DirSize(path string) (size int64, err error) {
	size = 0
	err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

//  Gets a list of tenant directories and their disk space consumption
func (c *Stats) Index() revel.Result {

	baseDir := models.GetOutputDirectory()

	// get a list of subdirectories
	files, err := ioutil.ReadDir(baseDir)
	if err != nil {
		panic(err)
	}

	// create a temp slice to hold the entries
	dirInfos := make([]DirStatEntry, len(files), len(files))

	for i, f := range files {
		fileName := f.Name()
		fullPath := path.Join(baseDir, fileName)

		// calculate the size of the directory
		dirSize, err := DirSize(fullPath)
		if err != nil {
			return c.RenderError(err)
		}

		tenant, err := models.TenantFromHomeDirectory(Dbm, fileName)
		var tenantName = "<UNNKNOWN>"
		// get the tenant name if we have a tenant
		if err == nil {
			tenantName = tenant.Username
		}

		dirInfos[i] = DirStatEntry{
			Name:   fileName,
			Size:   dirSize,
			Tenant: tenantName,
		}
	}
	// render the stats
	return c.RenderJson(StatsOutput{
		Directories: dirInfos,
	})
}
