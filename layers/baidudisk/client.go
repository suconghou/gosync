package baidudisk

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/suconghou/utilgo"
)

const name = "百度网盘"

// Bclient is a baidudisk client
type Bclient struct {
	token     string
	root      string
	path      string
	apiURL    string
	uploadURL string
	taskURL   string
	infoURL   string
}

type counter struct {
	readed   int64
	total    int64
	origin   *bytes.Buffer
	progress func(received int64, readed int64, total int64, start int64, end int64)
}

var taskStatusMap = map[string]string{
	"0": "下载成功",
	"1": "下载进行中",
	"2": "系统错误",
	"3": "资源不存在",
	"4": "下载超时",
	"5": "资源存在但下载失败",
	"6": "存储空间不足",
	"7": "任务已取消",
}

// Log is a global logger
var Log = log.New(os.Stdout, "", 0)

// NewClient return a client
func NewClient(token string, root string) *Bclient {
	return &Bclient{
		token:     token,
		root:      root,
		apiURL:    "https://pcs.baidu.com/rest/2.0/pcs/file",
		infoURL:   "https://pcs.baidu.com/rest/2.0/pcs/quota",
		uploadURL: "https://c.pcs.baidu.com/rest/2.0/pcs/file",
		taskURL:   "https://pan.baidu.com/rest/2.0/services/cloud_dl",
	}
}

func (c *counter) Read(p []byte) (int, error) {
	n, err := c.origin.Read(p)
	if err != nil {
		return n, err
	}
	if c.progress != nil {
		c.readed += int64(n)
		c.progress(c.readed, c.readed, c.total, 0, c.total)
	}
	return n, err
}

// Pwd print current dir
func (bc *Bclient) Pwd(p string) error {
	Log.Print(name + bc.root + "  ➜  " + p)
	return nil
}

// Ls print dir content in cli
func (bc *Bclient) Ls(p string) error {
	js, err := bc.APILs(p)
	if err != nil {
		return err
	}
	b := bytes.Buffer{}
	var total uint64
	for i := range js.Get("list").MustArray() {
		item := js.Get("list").GetIndex(i)
		size := item.Get("size").MustUint64()
		total = total + size
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("%-22s", utilgo.DateFormat(item.Get("ctime").MustInt64())))
		b.WriteString(fmt.Sprintf("%-22s", utilgo.DateFormat(item.Get("mtime").MustInt64())))
		b.WriteString(fmt.Sprintf("%-10s", utilgo.ByteFormat(size)))
		b.WriteString(fmt.Sprintf("%-20s", item.Get("path").MustString()))
	}
	Log.Printf("%s%s", name+bc.root+"  ➜  "+p+" "+utilgo.ByteFormat(total), b.String())
	return nil
}

// APILsURL return ls url string
func (bc *Bclient) APILsURL(p string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&path=%s", bc.apiURL, "list", bc.token, path.Join(bc.root, p))
}

// APILs response ls
func (bc *Bclient) APILs(p string) (*simplejson.Json, error) {
	body, err := utilgo.GetContent(bc.APILsURL(p), 10)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// Cd show files list
func (bc *Bclient) Cd(p string) error {
	bc.path = p
	return bc.Ls(p)
}

// Mkdir mkdir a dir
func (bc *Bclient) Mkdir(p string) error {
	js, err := bc.APIMkdir(p)
	if err != nil {
		return err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	Log.Printf("%s\n已创建 %s", name+bc.root, p)
	return nil
}

// APIMkdirURL return mkdir api url
func (bc *Bclient) APIMkdirURL(p string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&path=%s", bc.apiURL, "mkdir", bc.token, path.Join(bc.root, p))
}

// APIMkdir return api resp
func (bc *Bclient) APIMkdir(p string) (*simplejson.Json, error) {
	body, err := utilgo.PostContent(bc.APIMkdirURL(p), "application/x-www-form-urlencoded", nil, nil)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// Mv move files
func (bc *Bclient) Mv(source string, target string) error {
	js, err := bc.APIMv(source, target)
	if err != nil {
		return err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	Log.Printf("%s\n%s 已移动至 %s", name+bc.root, source, target)
	return nil
}

// APIMvURL return mv api url
func (bc *Bclient) APIMvURL(source string, target string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&from=%s&to=%s", bc.apiURL, "move", bc.token, path.Join(bc.root, source), path.Join(bc.root, target))
}

// APIMv return mv resp
func (bc *Bclient) APIMv(source string, target string) (*simplejson.Json, error) {
	body, err := utilgo.PostContent(bc.APIMvURL(source, target), "application/x-www-form-urlencoded", nil, nil)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// Cp copy files
func (bc *Bclient) Cp(source string, target string) error {
	js, err := bc.APICp(source, target)
	if err != nil {
		return err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	Log.Printf("%s\n%s 已复制至 %s", name+bc.root, source, target)
	return nil
}

// APICpURL return cp url
func (bc *Bclient) APICpURL(source string, target string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&from=%s&to=%s", bc.apiURL, "copy", bc.token, path.Join(bc.root, source), path.Join(bc.root, target))
}

// APICp return cp resp
func (bc *Bclient) APICp(source string, target string) (*simplejson.Json, error) {
	body, err := utilgo.PostContent(bc.APICpURL(source, target), "application/x-www-form-urlencoded", nil, nil)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// Rm delete files
func (bc *Bclient) Rm(file string) error {
	js, err := bc.APIRm(file)
	if err != nil {
		return err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	Log.Printf("%s\n%s 已删除", name+bc.root, file)
	return nil
}

// APIRmURL return rm api url
func (bc *Bclient) APIRmURL(file string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&path=%s", bc.apiURL, "delete", bc.token, path.Join(bc.root, file))
}

// APIRm return rm resp
func (bc *Bclient) APIRm(file string) (*simplejson.Json, error) {
	body, err := utilgo.PostContent(bc.APIRmURL(file), "application/x-www-form-urlencoded", nil, nil)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// Get return file reader
func (bc *Bclient) Get(file string) (io.ReadCloser, error) {
	return bc.APIGet(file)
}

// APIGet return file reader
func (bc *Bclient) APIGet(file string) (io.ReadCloser, error) {
	resp, err := utilgo.GetResp(bc.GetDownloadURL(file), 10)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// GetDownloadURL return download url
func (bc *Bclient) GetDownloadURL(file string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&path=%s", bc.apiURL, "download", bc.token, path.Join(bc.root, file))
}

// Put upload files may use rapid upload
func (bc *Bclient) Put(savePath string, overwrite bool, file *os.File) error {
	info, err := file.Stat()
	var (
		size = info.Size()
	)
	if size > 262144 { // 尝试秒传
		contentMd5, contentCrc32, sliceMd5, err := bc.RapidPut(file, savePath, overwrite)
		Log.Printf("文件大小: %d\n文件哈希: %s %s\n片段哈希: %s", size, contentMd5, contentCrc32, sliceMd5)
		if err == nil {
			Log.Printf("%s 已秒传", savePath)
			return err
		}
	}
	js, err := bc.APIPut(savePath, overwrite, file, size, utilgo.ProgressBar("", "", nil, os.Stdout))
	if err != nil {
		return err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	Log.Print(fmt.Sprintf("%s %s %d\n已上传", js.Get("path").MustString(), js.Get("md5").MustString(), js.Get("size").MustInt()))
	return nil
}

// APIPutURL return upload url
func (bc *Bclient) APIPutURL(savePath string, overwrite bool) string {
	ondup := "newcopy"
	if overwrite {
		ondup = "overwrite"
	}
	return fmt.Sprintf("%s?method=%s&access_token=%s&path=%s&ondup=%s", bc.uploadURL, "upload", bc.token, path.Join(bc.root, savePath), ondup)
}

// APIPut return put resp
func (bc *Bclient) APIPut(savePath string, overwrite bool, file *os.File, filesize int64, progress func(received int64, readed int64, total int64, start int64, end int64)) (*simplejson.Json, error) {
	var (
		r               io.Reader
		bodyBuf         = &bytes.Buffer{}
		bodyWriter      = multipart.NewWriter(bodyBuf)
		fileWriter, err = bodyWriter.CreateFormFile("file", path.Base(savePath))
	)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return nil, err
	}
	bodyWriter.Close()
	if progress != nil {
		r = &counter{origin: bodyBuf, total: int64(bodyBuf.Len()), progress: progress}
	} else {
		r = bodyBuf
	}
	body, err := utilgo.PostContent(bc.APIPutURL(savePath, overwrite), bodyWriter.FormDataContentType(), r, nil)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// RapidPut upload files
func (bc *Bclient) RapidPut(file *os.File, savePath string, overwrite bool) (string, string, string, error) {
	contentMd5, contentCrc32, sliceMd5, js, err := bc.APIRapidPut(file, savePath, overwrite)
	if err != nil {
		return contentMd5, contentCrc32, sliceMd5, err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return contentMd5, contentCrc32, sliceMd5, fmt.Errorf(errMsg)
	}
	return contentMd5, contentCrc32, sliceMd5, nil
}

// APIRapidPutURL return rapid upload url
func (bc *Bclient) APIRapidPutURL(savePath string, fileSize int64, md5Str string, sliceMd5 string, contentCrc32 string, overwrite bool) string {
	ondup := "newcopy"
	if overwrite {
		ondup = "overwrite"
	}
	return fmt.Sprintf("%s?method=%s&access_token=%s&path=%s&content-length=%d&content-md5=%s&slice-md5=%s&content-crc32=%s&ondup=%s", bc.uploadURL, "rapidupload", bc.token, path.Join(bc.root, savePath), fileSize, md5Str, sliceMd5, contentCrc32, ondup)
}

// APIRapidPut return RapidPut resp
func (bc *Bclient) APIRapidPut(file *os.File, savePath string, overwrite bool) (string, string, string, *simplejson.Json, error) {
	info, err := file.Stat()
	if err != nil {
		return "", "", "", nil, err
	}
	fileSize := info.Size()
	md5byte, err := utilgo.GetFileHash(file, "md5")
	if err != nil {
		return "", "", "", nil, err
	}
	contentMd5 := hex.EncodeToString(md5byte)
	crc32byte, err := utilgo.GetFileHash(file, "crc32")
	if err != nil {
		return "", "", "", nil, err
	}
	contentCrc32 := hex.EncodeToString(crc32byte)
	slice := make([]byte, 262144)
	file.ReadAt(slice, 0)
	sliceMd5 := fmt.Sprintf("%x", md5.Sum(slice))
	body, err := utilgo.PostContent(bc.APIRapidPutURL(savePath, fileSize, contentMd5, sliceMd5, contentCrc32, overwrite), "application/x-www-form-urlencoded", nil, nil)
	defer file.Seek(0, 0)
	if err != nil {
		return "", "", "", nil, err
	}
	js, err := simplejson.NewJson(body)
	return contentMd5, contentCrc32, sliceMd5, js, err
}

// Info print the disk usage
func (bc *Bclient) Info() error {
	js, err := bc.APIInfo()
	if err != nil {
		return err
	}
	b := bytes.Buffer{}
	quota := js.Get("quota").MustUint64()
	used := js.Get("used").MustUint64()
	b.WriteString(name + "\n总大小:" + utilgo.ByteFormat(quota))
	b.WriteString("\n已使用:" + utilgo.ByteFormat(used))
	b.WriteString(fmt.Sprintf("\n利用率:%.1f%%", float32(used)/float32(quota)*100))
	Log.Print(b.String())
	return nil
}

// APIInfoURL return disk info url
func (bc *Bclient) APIInfoURL() string {
	return fmt.Sprintf("%s?method=%s&access_token=%s", bc.infoURL, "info", bc.token)
}

// APIInfo response usage info
func (bc *Bclient) APIInfo() (*simplejson.Json, error) {
	body, err := utilgo.GetContent(bc.APIInfoURL(), 10)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// FileInfo print the file/dir info
func (bc *Bclient) FileInfo(p string, dlink bool) error {
	js, err := bc.APIFileInfo(p)
	if err != nil {
		return err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	b := bytes.Buffer{}
	item := js.Get("list").GetIndex(0)
	b.WriteString(name + item.Get("path").MustString())
	b.WriteString("\n文件类型:" + utilgo.BoolString(item.Get("isdir").MustInt() == 0, "文件", "文件夹"))
	b.WriteString("\n文件大小:" + utilgo.ByteFormat(item.Get("size").MustUint64()))
	b.WriteString(fmt.Sprintf("\n文件字节:%d\n文件标识:%d", item.Get("size").MustInt64(), item.Get("fs_id").MustInt64()))
	b.WriteString("\n创建时间:" + utilgo.DateFormat(item.Get("ctime").MustInt64()))
	b.WriteString("\n修改时间:" + utilgo.DateFormat(item.Get("mtime").MustInt64()))
	blockstr := item.Get("block_list").MustString()
	if blockstr != "" {
		blocks, err := simplejson.NewJson([]byte(item.Get("block_list").MustString()))
		if err != nil {
			Log.Print(b.String())
			return err
		}
		blocksarr := blocks.MustStringArray()
		if len(blocksarr) == 1 {
			b.WriteString("\n文件哈希:" + blocksarr[0])
		} else {
			for _, v := range blocksarr {
				b.WriteString("\n哈希块:" + v)
			}
		}
	}
	if dlink {
		b.WriteString("\n下载地址:" + bc.GetDownloadURL(p))
	}
	Log.Print(b.String())
	return nil
}

// APIFileInfoURL return fileinfo url
func (bc *Bclient) APIFileInfoURL(file string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&path=%s", bc.apiURL, "meta", bc.token, path.Join(bc.root, file))
}

// APIFileInfo response info
func (bc *Bclient) APIFileInfo(file string) (*simplejson.Json, error) {
	body, err := utilgo.GetContent(bc.APIFileInfoURL(file), 10)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// Search search files
func (bc *Bclient) Search(fileName string) error {
	js, err := bc.APISearch(fileName)
	if err != nil {
		return err
	}
	b := bytes.Buffer{}
	var total uint64
	for i := range js.Get("list").MustArray() {
		item := js.Get("list").GetIndex(i)
		size := item.Get("size").MustUint64()
		total = total + size
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("%-22s", utilgo.DateFormat(item.Get("ctime").MustInt64())))
		b.WriteString(fmt.Sprintf("%-22s", utilgo.DateFormat(item.Get("mtime").MustInt64())))
		b.WriteString(fmt.Sprintf("%-10s", utilgo.ByteFormat(size)))
		b.WriteString(fmt.Sprintf("%-20s", item.Get("path").MustString()))
	}
	Log.Printf("%s\n%s", name+bc.root+"  ➜  搜索["+fileName+"] "+utilgo.ByteFormat(total), b.String())
	return nil
}

// APISearchURL return api search url
func (bc *Bclient) APISearchURL(name string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&path=%s&wd=%s&re=%s", bc.apiURL, "search", bc.token, bc.root, name, "1")
}

// APISearch return search resp
func (bc *Bclient) APISearch(name string) (*simplejson.Json, error) {
	body, err := utilgo.GetContent(bc.APISearchURL(name), 10)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// TaskAdd add task
func (bc *Bclient) TaskAdd(savePath string, url string) error {
	js, err := bc.APITaskAdd(savePath, url)
	if err != nil {
		return err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	id := js.Get("task_id").MustInt()
	Log.Printf("任务ID %d\n", id)
	if js.Get("rapid_download").MustInt() == 1 {
		Log.Print("离线已秒杀")
		bc.TaskInfo(strconv.Itoa(id))
	}
	return nil
}

// APITaskAddURL retrun taskadd url
func (bc *Bclient) APITaskAddURL(savePath string, sourceURL string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&save_path=%s&source_url=%s&app_id=250528", bc.taskURL, "add_task", bc.token, savePath, sourceURL)
}

// APITaskAdd return taskadd resp
func (bc *Bclient) APITaskAdd(savePath string, sourceURL string) (*simplejson.Json, error) {
	savePath = path.Join(bc.root, savePath)
	body, err := utilgo.PostContent(bc.APITaskAddURL(savePath, sourceURL), "application/x-www-form-urlencoded", nil, nil)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// TaskList get tasklist
func (bc *Bclient) TaskList() error {
	js, err := bc.APITaskList()
	if err != nil {
		return err
	}
	b := bytes.Buffer{}
	for i := range js.Get("task_info").MustArray() {
		item := js.Get("task_info").GetIndex(i)
		id := item.Get("task_id").MustString()
		name := item.Get("task_name").MustString()
		t, _ := strconv.ParseInt(item.Get("create_time").MustString(), 10, 64)
		createTime := utilgo.DateFormat(t)
		status := showTaskStatus(item.Get("status").MustString())
		sourceURL := item.Get("source_url").MustString()
		savePath := item.Get("save_path").MustString()
		b.WriteString(fmt.Sprintf("\n任务ID:%s\n任务名称:%s\n创建时间:%s\n任务状态:%s\n源地址:%s \n存储至:%s\n", id, name, createTime, status, sourceURL, savePath))
	}
	Log.Printf("%s%s  ➜  离线任务: %d个任务 %s", name, bc.root, js.Get("total").MustInt(), b.String())
	return nil
}

// APITaskListURL return tasklist url
func (bc *Bclient) APITaskListURL() string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&status=255&app_id=250528&need_task_info=1", bc.taskURL, "list_task", bc.token)
}

// APITaskList retrun tasklist resp
func (bc *Bclient) APITaskList() (*simplejson.Json, error) {
	body, err := utilgo.PostContent(bc.APITaskListURL(), "application/x-www-form-urlencoded", nil, nil)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// TaskInfo get taskinfo
func (bc *Bclient) TaskInfo(ids string) error {
	js, err := bc.APITaskInfo(ids)
	if err != nil {
		return err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	b := bytes.Buffer{}
	timestamp := time.Now().Unix()
	for id := range js.Get("task_info").MustMap() {
		item := js.Get("task_info").Get(id)
		name := item.Get("task_name").MustString()
		status := showTaskStatus(item.Get("status").MustString())
		createTime, _ := strconv.ParseInt(item.Get("create_time").MustString(), 10, 64)
		startTime, _ := strconv.ParseInt(item.Get("create_time").MustString(), 10, 64)
		b.WriteString(fmt.Sprintf("\n任务ID:%s\n任务名称:%s\n任务状态:%s\n创建时间:%s\n开始下载时间:%s\n", id, name, status, utilgo.DateFormat(createTime), utilgo.DateFormat(startTime)))
		fileSize, _ := strconv.ParseUint(item.Get("file_size").MustString(), 10, 64)
		finishTime, _ := strconv.ParseInt(item.Get("finish_time").MustString(), 10, 64)
		if fileSize > 0 { //已探测出文件大小
			b.WriteString(fmt.Sprintf("大小:%d (%s)\n", fileSize, utilgo.ByteFormat(fileSize)))
			if finishTime > startTime { //已下载完毕
				duration := finishTime - startTime
				b.WriteString(fmt.Sprintf("任务完成时间:%s 耗时:%d秒 速度:%.2fKB/s \n", utilgo.DateFormat(finishTime), duration, float64(fileSize)/1024/float64(duration)))
			} else if finishTime > 0 && finishTime == startTime {
				b.WriteString(fmt.Sprintf("任务完成时间:%s 云端已秒杀 \n", utilgo.DateFormat(finishTime)))
			} else {
				finishedSize, _ := strconv.ParseUint(item.Get("finished_size").MustString(), 10, 64)
				duration := int64(timestamp) - startTime
				b.WriteString(fmt.Sprintf("已下载:%s 进度:%.1f%% 速度:%.2fKB/s\n", utilgo.ByteFormat(finishedSize), float64(finishedSize)/float64(fileSize)*100, float64(finishedSize)/1024/float64(duration)))
			}
		}
		b.WriteString(fmt.Sprintf("原地址:%s\n存储至:%s\n", item.Get("source_url").MustString(), item.Get("save_path").MustString()))
	}
	Log.Printf("%s%s  ➜  任务详情: %s", name, bc.root, b.String())
	return nil
}

// APITaskInfoURL return taskinfo url
func (bc *Bclient) APITaskInfoURL(ids string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&task_ids=%s&app_id=250528", bc.taskURL, "query_task", bc.token, ids)
}

// APITaskInfo return taskinfo resp
func (bc *Bclient) APITaskInfo(ids string) (*simplejson.Json, error) {
	body, err := utilgo.PostContent(bc.APITaskInfoURL(ids), "application/x-www-form-urlencoded", nil, nil)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// TaskRemove remove task
func (bc *Bclient) TaskRemove(id string) error {
	js, err := bc.APITaskRemove(id)
	if err != nil {
		return err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	Log.Printf("已取消任务%s\n", id)
	return nil
}

// APITaskRemoveURL return taskremove url
func (bc *Bclient) APITaskRemoveURL(id string) string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&task_id=%s&app_id=250528", bc.taskURL, "cancel_task", bc.token, id)
}

// APITaskRemove return taskremove resp
func (bc *Bclient) APITaskRemove(id string) (*simplejson.Json, error) {
	body, err := utilgo.PostContent(bc.APITaskRemoveURL(id), "application/x-www-form-urlencoded", nil, nil)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

// Clear empty recycle
func (bc *Bclient) Clear() error {
	js, err := bc.APIClear()
	if err != nil {
		return err
	}
	errMsg := js.Get("error_msg").MustString()
	if errMsg != "" {
		return fmt.Errorf(errMsg)
	}
	Log.Print("已清空回收站 " + strconv.Itoa(js.GetPath("extra.succnum").MustInt()) + "个项目被清除")
	return nil
}

// APIClearURL return clear url
func (bc *Bclient) APIClearURL() string {
	return fmt.Sprintf("%s?method=%s&access_token=%s&type=%s", bc.apiURL, "delete", bc.token, "recycle")
}

// APIClear return clear resp
func (bc *Bclient) APIClear() (*simplejson.Json, error) {
	body, err := utilgo.PostContent(bc.APIClearURL(), "application/x-www-form-urlencoded", nil, nil)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(body)
}

func showTaskStatus(status string) string {
	if v, ok := taskStatusMap[status]; ok {
		return v
	}
	return "未知状态"
}
