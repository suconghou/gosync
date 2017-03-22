package commands

import (
	"flag"
	"fmt"
	"net/http"
	"netdisk/config"
	"netdisk/layers/fslayer"
	"netdisk/util"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func Use() {
	fmt.Println("select default remote connect")
	fmt.Println(config.Cfg)
}

func Ls() {
	var path string
	if len(os.Args) == 3 {
		path = absNoRoot(os.Args[2])
	} else {
		path = config.Cfg.Path
	}
	fslayer.ListDir(path)
}

func Cd() {
	if len(os.Args) == 3 {
		config.Cfg.Path = absNoRoot(os.Args[2])
		ret := fslayer.ListDir(config.Cfg.Path)
		if ret {
			config.SaveConfig()
		}
	} else {
		fmt.Println("Usage:disk cd newpath")
	}
}

func Pwd() {
	fmt.Println(util.DiskName(config.Cfg.Disk) + config.Cfg.Root + "  ➜  " + config.Cfg.Path)
}

func Mv() {
	if len(os.Args) == 4 {
		var source string = absPath(os.Args[2])
		var target string = absPath(os.Args[3])
		ok, _, _ := fslayer.GetFileInfo(source, false)
		if ok {
			fslayer.MoveFile(source, target)
		}
	} else {
		fmt.Println("Usage:disk mv path newpath")
	}
}

func Cp() {
	if len(os.Args) == 4 {
		var source string = absPath(os.Args[2])
		var target string = absPath(os.Args[3])
		ok, _, _ := fslayer.GetFileInfo(source, false)
		if ok {
			fslayer.CopyFile(source, target)
		}
	} else {
		fmt.Println("Usage:disk cp path newpath")
	}
}

func Mkdir() {

	if len(os.Args) == 3 {
		var path string = absPath(os.Args[2])
		fslayer.Mkdir(path)
	} else {
		fmt.Println("Usage:disk mkdir path")
	}

}

func Rm() {
	if len(os.Args) == 3 {
		var path string = absPath(os.Args[2])
		ok, _, _ := fslayer.GetFileInfo(path, false)
		if ok {
			fslayer.DeleteFile(path)
		}
	} else {

		fmt.Println("Usage:disk rm filepath")
	}
}

func Get() {
	if len(os.Args) >= 3 {
		var filePath = absPath(os.Args[2])

		var dist string = ""
		if len(os.Args) >= 4 {
			dist = absLocalPath(os.Args[3])
		} else {
			dist = absLocalPath(path.Base(filePath))
		}
		ok, size, hash := fslayer.GetFileInfo(filePath, false)
		if ok {
			fslayer.Get(filePath, dist, size, hash)
		}
	} else {
		fmt.Println("Usage:disk get filepath saveas")
	}
}

func Put() {
	if len(os.Args) >= 3 {
		var path string = absLocalPath(os.Args[2])
		var savePath string = absPath(os.Args[2])
		fileSize, md5Str := util.FileOk(path)
		var ondup string = util.BoolString(len(os.Args) >= 4, "overwrite", "newcopy")
		if fileSize > 262144 {
			fslayer.PutFileRapid(path, savePath, fileSize, ondup, md5Str)
		} else if fileSize > 1 {
			fslayer.PutFile(path, savePath, fileSize, ondup)
		} else {
			fmt.Println(path + "不存在或不可读")
		}
	} else {
		fmt.Println("Usage:disk put filepath saveas")
	}
}

func Wget() {
	if len(os.Args) >= 3 {
		var filePath = absPath(os.Args[2])
		var dist string = ""
		if len(os.Args) >= 4 && (!strings.Contains(os.Args[3], "-")) {
			dist = absLocalPath(os.Args[3])
		} else {
			dist = absLocalPath(path.Base(filePath))
		}
		if strings.HasPrefix(os.Args[2], "http://") || strings.HasPrefix(os.Args[2], "https://") {
			tokens := strings.Split(dist, "?")
			fslayer.WgetUrl(os.Args[2], tokens[0])
		} else {
			ok, size, hash := fslayer.GetFileInfo(filePath, false)
			if ok {
				fslayer.Wget(filePath, dist, size, hash)
			}
		}

	} else {
		fmt.Println("Usage:disk wget filepath/url saveas")
	}
}

func Sync() {

}

func Info() {
	if len(os.Args) == 2 {
		fslayer.GetInfo()
		fmt.Println("配置文件:" + config.ConfigPath)
	} else {
		fslayer.GetFileInfo(absPath(os.Args[2]), false)
	}
}

func Hash() {
	if len(os.Args) == 3 {
		var filePath string = absLocalPath(os.Args[2])
		util.PrintMd5(filePath)
	} else {
		fmt.Println("Usage:disk hash file")
	}
}

func Play() {
	if len(os.Args) >= 3 {
		var filePath = absPath(os.Args[2])
		var dist string = ""
		if len(os.Args) >= 4 && (!strings.Contains(os.Args[3], "-")) {
			dist = absLocalPath(os.Args[3])
		} else {
			dist = absLocalPath(path.Base(filePath))
		}
		stdout := util.HasFlag("--stdout")
		if strings.HasPrefix(os.Args[2], "http://") || strings.HasPrefix(os.Args[2], "https://") {
			tokens := strings.Split(dist, "?")
			fslayer.PlayUrl(os.Args[2], tokens[0], stdout)
		} else {
			ok, size, hash := fslayer.GetFileInfo(filePath, stdout)
			if ok {
				fslayer.GetPlayStream(filePath, dist, size, hash, stdout)
			}
		}
	} else {
		fmt.Println("Usage:disk play filepath/url")
	}
}

func Help() {
	fmt.Println(os.Args[0] + " ls info mv get put wget play rm mkdir pwd hash config search task ")
}

func Config() {
	if (len(os.Args) == 2) || (os.Args[2] == "list") {
		config.ConfigList()
	} else if len(os.Args) == 3 && os.Args[2] == "get" {
		config.ConfigGet()
	} else if len(os.Args) == 4 && os.Args[2] == "set" {
		config.ConfigSet(os.Args[3])
	} else if len(os.Args) == 4 && os.Args[2] == "setapp" {
		config.ConfigSetApp(os.Args[3])
	} else {
		fmt.Println("Usage:disk config list/get/set/setapp")
	}
}

func Task() {
	if (len(os.Args) == 2) || (os.Args[2] == "list") {
		fslayer.GetTaskList()
	} else if len(os.Args) == 5 && os.Args[2] == "add" {
		fslayer.AddTask(absPath(os.Args[3]), os.Args[4])
	} else if len(os.Args) == 4 && os.Args[2] == "remove" {
		fslayer.RemoveTask(os.Args[3])
	} else if len(os.Args) == 4 && os.Args[2] == "info" {
		fslayer.GetTaskInfo(os.Args[3])
	} else {
		fmt.Println("Usage:disk task list/add/info/remove")
	}
}

func Search() {
	if len(os.Args) == 3 {
		fslayer.SearchFile(os.Args[2])
	}
}

func Empty() {
	fslayer.Empty()
}

func Serve() {
	var (
		port int
		root string
	)
	var ferr flag.ErrorHandling
	var CommandLine = flag.NewFlagSet(os.Args[0], ferr)
	CommandLine.IntVar(&port, "p", 6060, "http server port")
	CommandLine.StringVar(&root, "d", "./", "root document dir")
	err := CommandLine.Parse(os.Args[2:])
	if err != nil {
		os.Exit(2)
	}
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var doc string
	if path.IsAbs(root) {
		doc = root
	} else {
		doc = path.Join(pwd, root)
	}
	if _, err := os.Stat(doc); err == nil {
		fmt.Println("Server listening on port " + strconv.Itoa(port))
		fmt.Println("Document root " + doc)
		error := http.ListenAndServe(":"+strconv.Itoa(port), http.FileServer(http.Dir(doc)))
		if error != nil {
			fmt.Println("Error listening:", error.Error())
			os.Exit(1)
		}
	} else {
		fmt.Println(doc + " not exists")
	}

}

func Usage() {
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		fmt.Println(os.Args[0] + " version: disk/" + config.Version + "\n" + config.ReleaseUrl)
	} else {
		fmt.Println(os.Args[0] + " ls info mv cp get put wget play rm mkdir pwd hash config empty")
	}
}

func Daemon() {
	fmt.Println("daemon start")
}

func absPath(filePath string) string {
	filePath = path.Clean(filePath)
	if path.IsAbs(filePath) {
		if !strings.HasPrefix(filePath, config.Cfg.Root) {
			filePath = fmt.Sprintf("%s/%s", config.Cfg.Root, "."+filePath)
		}
	} else {
		filePath = fmt.Sprintf("%s/%s/%s", config.Cfg.Root, config.Cfg.Path, filePath)
	}
	return path.Clean(filePath)
}

func absNoRoot(filePath string) string {
	filePath = path.Clean(filePath)
	if path.IsAbs(filePath) {
		filePath = filePath
	} else {
		filePath = fmt.Sprintf("%s/%s", config.Cfg.Path, filePath)
	}
	return path.Clean(filePath)
}

func absLocalPath(filePath string) string {
	filePath = filepath.Clean(filePath)
	if filepath.IsAbs(filePath) {
		filePath = filePath
	} else {
		dir, _ := os.Getwd()
		filePath = filepath.Join(dir, filepath.Base(filePath))
	}
	return filepath.Clean(filePath)
}
