package storage

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

const LINK_FILENAME_LENGTH = 64

func LinksPath(userHomeDirPath string) string {
	return filepath.Join(userHomeDirPath, MESSAGES_LINKS_DIRECTORY)
}
func LinkPath(userHomeDirPath, link string) string {
	return filepath.Join(LinksPath(userHomeDirPath), link)
}
func UserHasLink(userHomeDirPath, link string) (bool, error) {
	_, err := os.Stat(LinkPath(userHomeDirPath, link))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func StoreLink(userHomeDirPath, link string, contactData []byte) error {
	path := LinkPath(userHomeDirPath, link)
	err := ioutil.WriteFile(path, contactData, 0644)
	if err != nil {
		return err
	}
	return nil
}

func DeleteLink(userHomeDirPath, link string) error {
	linkPath := LinkPath(userHomeDirPath, link)
	if err := os.Remove(linkPath); err != nil {
		return err
	}
	return nil
}

func ListLinks(userHomeDirPath string) ([]string, error) {
	linksPath := LinksPath(userHomeDirPath)
	var matchingFiles []string

	linkFiles, err := ioutil.ReadDir(linksPath)
	if err != nil {
		return nil, err
	}

	for _, link := range linkFiles {
		if !link.IsDir() && len(link.Name()) == LINK_FILENAME_LENGTH {
			linkData, err := os.ReadFile(filepath.Join(linksPath, link.Name()))
			if err != nil {
				return matchingFiles, err
			}
			matchingFiles = append(matchingFiles, string(linkData))
		}
	}

	return matchingFiles, nil
}
