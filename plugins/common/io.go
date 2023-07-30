package common

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"strings"
)

func FileToSlice(filename string) (lines []string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		lines = append(lines, text)
	}

	err = scanner.Err()
	return
}

func CatFile(filename string) {
	slice, err := FileToSlice(filename)
	if err != nil {
		LogDebug(fmt.Sprintf("Error cat'ing file %s: %s", filename, err.Error()))
		return
	}

	for _, line := range slice {
		LogDebug(fmt.Sprintf("line: %s", line))
	}
}

func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}

	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}

	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return
	}

	err = out.Sync()
	return
}

func DirectoryExists(filename string) bool {
	fi, err := os.Stat(filename)
	if err != nil {
		return false
	}

	return fi.IsDir()
}

func FileExists(filename string) bool {
	fi, err := os.Stat(filename)
	if err != nil {
		return false
	}

	return fi.Mode().IsRegular()
}

func IsAbsPath(path string) bool {
	return strings.HasPrefix(path, "/")
}

func ListFilesWithPrefix(path string, prefix string) []string {
	names, err := ioutil.ReadDir(path)
	if err != nil {
		return []string{}
	}

	files := []string{}
	for _, f := range names {
		if prefix != "" && !strings.HasPrefix(f.Name(), prefix) {
			continue
		}

		if f.Mode().IsRegular() {
			files = append(files, fmt.Sprintf("%s%s", path, f.Name()))
		}
	}

	return files
}

func ReadFirstLine(filename string) (text string) {
	if !FileExists(filename) {
		return
	}
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if text = strings.TrimSpace(scanner.Text()); text == "" {
			continue
		}
		return
	}
	return
}

func SetPermissions(path string, fileMode os.FileMode) error {
	if err := os.Chmod(path, fileMode); err != nil {
		return err
	}

	systemGroup := GetenvWithDefault("CLAIR_SYSTEM_GROUP", "clair")
	systemUser := GetenvWithDefault("CLAIR_SYSTEM_USER", "clair")

	if strings.HasPrefix(path, "/etc/sudoers.d/") {
		systemGroup = "root"
		systemUser = "root"
	}

	group, err := user.LookupGroup(systemGroup)
	if err != nil {
		return err
	}
	user, err := user.Lookup(systemUser)
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(group.Gid)
	if err != nil {
		return err
	}

	return os.Chown(path, uid, gid)
}

func TouchFile(filename string) error {
	mode := os.FileMode(0600)
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer file.Close()

	file.Chmod(mode)
	SetPermissions(filename, mode)
	return nil
}

func WriteSliceToFile(filename string, lines []string) error {
	mode := os.FileMode(0600)
	if strings.HasPrefix(filename, "/etc/sudoers.d/") {
		mode = os.FileMode(0440)
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	if err = w.Flush(); err != nil {
		return err
	}

	file.Chmod(mode)

	return nil
}
