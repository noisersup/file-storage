package dirs

type File struct {
	hash   []byte
	parent *File
}

type DirTree struct {
	root  *File
	files []File
}

func (t *DirTree) GetFile(path string) error {
	//Gets directories in plain

	//Encrypt dirnames
	/*	for n, path := range arr {
		path, err := encrypt(path)
		if err != nil {
			return err
		}
	}*/
	//attach every dirname to File struct and set parent element
	return nil
}

/* I really like this part of code and its sad that its not useful...
func pathToArr(path string) []string {
	match := regexp.MustCompile("[^/]+").FindAllStringSubmatch(path, -1)

	var getFullPaths func(dirs [][]string, prevPath string) []string

	getFullPaths = func(dirs [][]string, prevPath string) []string {
		if len(dirs) == 0 {
			return []string{}
		}
		path := prevPath + "/" + dirs[0][0]
		return append([]string{path}, getFullPaths(dirs[1:], path)...)
	}

	return getFullPaths(match, "")
}
*/
