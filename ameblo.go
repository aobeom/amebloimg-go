package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// DLserver 并行控制
type DLserver struct {
	WG    sync.WaitGroup
	Gonum chan string
	Img   chan []string
}

// BlogInfo 博客基本信息
type BlogInfo struct {
	author string
	target string
	host   string
}

// getCurrentDirectory 获取当前路径
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

// pathExists 判断文件夹是否存在
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// removeDuplicate 去重
func removeDuplicate(s []string) (ret []string) {
	LEN := len(s)
	for i := 0; i < LEN; i++ {
		if (i > 0 && s[i-1] == s[i]) || len(s[i]) == 0 {
			continue
		}
		ret = append(ret, s[i])
	}
	return
}

// urlCheck 检查地址有效性
func urlCheck(url string) (valid string, ok bool) {
	if strings.Contains(url, "https://ameblo.jp/") {
		valid = url
		ok = true
	} else {
		valid = ""
		ok = false
	}
	return
}

// request 统一请求结构
func request(method string, url string, body io.Reader, para map[string]string, mode bool) (*http.Response, error) {
	client := http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest(method, url, body)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.94 Safari/537.36")
	// 带参数
	if mode {
		q := req.URL.Query()
		for key, value := range para {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}
	return client.Do(req)
}

// getEntriesID 获取所有entry的ID
func getEntriesID(entry string) (eid string) {
	regID := regexp.MustCompile("entry-[0-9]+")
	eid = strings.Split(regID.FindAllString(entry, -1)[0], "-")[1]
	return
}

// getBlogRaw 获取原始内容
func getBlogRaw(url string) *goquery.Document {
	para := make(map[string]string)
	model := false
	res, _ := request("GET", url, nil, para, model)
	defer res.Body.Close()
	// 返回goquery的结构
	body, _ := goquery.NewDocumentFromReader(res.Body)
	return body
}

// newNow 当前年月
func newNow() (today string) {
	// 200601是固定格式
	today = time.Now().Format("200601")
	return
}

// int2str 数字转字符串并补零
func int2str(i int) (s string) {
	S := strconv.Itoa(i)
	if len(S) == 1 {
		s = "0" + S
	} else {
		s = S
	}
	return
}

// timeCheck 检查时间格式有效性
func timeCheck(D string) (T string, S bool) {
	t, err := time.Parse("200601", D)
	if err == nil {
		T = t.Format("200601")
		S = true
	} else {
		T = ""
		S = false
	}
	return
}

//TimeGenarater 根据日期生成地址
func TimeGenarater(D string) (DATE []string) {
	datePart := strings.Split(D, "-")
	checkSep := len(datePart)
	if checkSep > 1 {
		// 只有结束时间
		if datePart[0] == "" {
			d, ok := timeCheck(datePart[1])
			if ok {
				// 提取年月数字
				year := d[0:4]
				month := d[4:]
				yearMAX, _ := strconv.Atoi(year)
				monthMAX, _ := strconv.Atoi(month)
				// 开始时间统一是amblo创立时间
				for yearMIN := 2005; yearMIN <= yearMAX; yearMIN++ {
					var newD string
					// 截止到指定的年月
					if yearMIN == yearMAX {
						for monthMIN := 1; monthMIN <= monthMAX; monthMIN++ {
							newD = int2str(yearMIN) + int2str(monthMIN)
							DATE = append(DATE, newD)
						}
						// 中间年份自动填充12个月
					} else {
						for monthMIN := 1; monthMIN <= 12; monthMIN++ {
							newD = int2str(yearMIN) + int2str(monthMIN)
							DATE = append(DATE, newD)
						}
					}
				}
			} else {
				fmt.Println("Time Error")
			}
			// 只有开始时间
		} else if datePart[1] == "" {
			d, ok := timeCheck(datePart[0])
			if ok {
				// 当前年月
				yearNow := newNow()[0:4]
				monthNow := newNow()[4:]
				yearMAX, _ := strconv.Atoi(yearNow)
				monthMAX, _ := strconv.Atoi(monthNow)
				// 指定年月
				year := d[0:4]
				month := d[4:]
				yearMIN, _ := strconv.Atoi(year)
				monthMIN, _ := strconv.Atoi(month)
				// 第一年从指定月份开始
				if yearMIN != yearMAX {
					for monthMIN <= 12 {
						var newD string
						newD = int2str(yearMIN) + int2str(monthMIN)
						DATE = append(DATE, newD)
						monthMIN++
					}
				} else {
					for monthMIN <= monthMAX {
						var newD string
						newD = int2str(yearMIN) + int2str(monthMIN)
						DATE = append(DATE, newD)
						monthMIN++
					}
				}
				// 第二年自动填充12个月
				yearMINnext := yearMIN + 1
				for yearMINnext <= yearMAX {
					var newD string
					// 截至到当前年月
					if yearMINnext == yearMAX {
						for monthMIN := 1; monthMIN <= monthMAX; monthMIN++ {
							newD = int2str(yearMINnext) + int2str(monthMIN)
							DATE = append(DATE, newD)
						}
						// 中间年份自动填充12个月
					} else {
						for monthMIN := 1; monthMIN <= 12; monthMIN++ {
							newD = int2str(yearMINnext) + int2str(monthMIN)
							DATE = append(DATE, newD)
						}
					}
					yearMINnext++
				}
			} else {
				fmt.Println("Time Error")
			}
		} else {
			// 指定开始和结束时间
			startT, sOK := timeCheck(datePart[0])
			endT, eOK := timeCheck(datePart[1])
			if startT <= endT {
				if sOK && eOK {
					// 提取起止时间的数字
					yearSta := startT[0:4]
					monthSta := startT[4:]
					yearEnd := endT[0:4]
					monthEnd := endT[4:]
					yearMIN, _ := strconv.Atoi(yearSta)
					monthMIN, _ := strconv.Atoi(monthSta)
					yearMAX, _ := strconv.Atoi(yearEnd)
					monthMAX, _ := strconv.Atoi(monthEnd)
					// 第一年从指定月开始
					if yearMIN != yearMAX {
						for monthMIN <= 12 {
							var newD string
							newD = int2str(yearMIN) + int2str(monthMIN)
							DATE = append(DATE, newD)
							monthMIN++
						}
					} else {
						for monthMIN <= monthMAX {
							var newD string
							newD = int2str(yearMIN) + int2str(monthMIN)
							DATE = append(DATE, newD)
							monthMIN++
						}
					}
					// 第二年开始自动填充12个月
					yearMINnext := yearMIN + 1
					for yearMINnext <= yearMAX {
						var newD string
						// 截至到当前年月
						if yearMINnext == yearMAX {
							for monthMIN := 1; monthMIN <= monthMAX; monthMIN++ {
								newD = int2str(yearMINnext) + int2str(monthMIN)
								DATE = append(DATE, newD)
							}
							// 中间年份自动填充12个月
						} else {
							for monthMIN := 1; monthMIN <= 12; monthMIN++ {
								newD = int2str(yearMINnext) + int2str(monthMIN)
								DATE = append(DATE, newD)
							}
						}
						yearMINnext++
					}
				} else {
					fmt.Println("Time Error")
				}
			} else {
				fmt.Println("Time Error")
			}
		}
	} else if checkSep == 1 {
		d, ok := timeCheck(datePart[0])
		if ok {
			DATE = append(DATE, d)
		} else {
			fmt.Println("Time Error")
		}
	} else {
		fmt.Println("Time Error")
	}
	return
}

// GetFirstPageEntry 获取第1页的entry id
func GetFirstPageEntry(url string) (eids []string) {
	blogRaw := getBlogRaw(url)
	blogRaw.Find("#imgList .imgTitle").Each(func(i int, s *goquery.Selection) {
		url, _ := s.Find("a").Attr("href")
		eid := getEntriesID(url)
		eids = append(eids, eid)
	})
	return
}

// GetOtherPageEntryByAPI 通过API获取第1页之后的entry id
func GetOtherPageEntryByAPI(blogInfo *BlogInfo) (eids []string) {
	entryAPIURL := "https://blogimgapi.ameba.jp/image_list/get.jsonp?"
	para := make(map[string]string)
	mode := true
	para["ameba_id"] = blogInfo.author
	para["limit"] = "18"
	para["target_ym"] = blogInfo.target
	para["sp"] = "false"
	for p := 2; p <= 4; p++ {
		para["page"] = strconv.Itoa(p)
		res, _ := request("GET", entryAPIURL, nil, para, mode)
		body, _ := ioutil.ReadAll(res.Body)
		oriData := string(body)
		// 移除多余字符串用于json格式
		oriData = strings.Replace(oriData, "Amb.Ameblo.image.Callback(", "", -1)
		oriData = strings.Replace(oriData, ");", "", -1)
		// json格式化
		data := make(map[string]interface{})
		json.Unmarshal([]byte(oriData), &data)
		// 获取entry id
		if data["success"].(bool) {
			entryList := data["imgList"].([]interface{})
			for _, entry := range entryList {
				entryURL := entry.(map[string]interface{})["entryUrl"]
				eid := getEntriesID(entryURL.(string))
				eids = append(eids, eid)
			}
		} else {
			eids = []string{}
		}
	}
	return
}

// GetImgURLByAPI 通过API获取图片地址
func GetImgURLByAPI(blogInfo *BlogInfo, eid string) (imgURLs []string) {
	imgHost := "http://stat.ameba.jp"
	imgAPIURL := "https://blogimgapi.ameba.jp/read_ahead/get.jsonp?"
	para := make(map[string]string)
	mode := true
	para["ameba_id"] = blogInfo.author
	para["old"] = "true"
	para["sp"] = "false"
	para["entry_id"] = eid
	res, _ := request("GET", imgAPIURL, nil, para, mode)
	body, _ := ioutil.ReadAll(res.Body)
	oriData := string(body)
	oriData = strings.Replace(oriData, "Amb.Ameblo.image.Callback(", "", -1)
	oriData = strings.Replace(oriData, ");", "", -1)
	data := make(map[string]interface{})
	json.Unmarshal([]byte(oriData), &data)
	regOrig := regexp.MustCompile(`t[0-9]+\_`)
	if data["success"].(bool) {
		entryList := data["imgList"].([]interface{})
		for _, entry := range entryList {
			imgURI := entry.(map[string]interface{})["imgUrl"]
			imgURIOrig := regOrig.ReplaceAllString(imgURI.(string), "o")
			imgURL := imgHost + imgURIOrig
			imgURLs = append(imgURLs, imgURL)
		}
	} else {
		imgURLs = []string{}
	}
	sort.Strings(imgURLs)
	imgURLs = removeDuplicate(imgURLs)
	return
}

// getImgURLEngine 图片过滤引擎
func getImgURLEngine(id int, e string, dl *DLserver, blogInfo *BlogInfo) {
	log.Printf("<Blog ID %d> 解析图片地址\n", id)
	var imgurls []string
	img := GetImgURLByAPI(blogInfo, e)
	imgurls = append(imgurls, img...)
	dl.WG.Done()
	<-dl.Gonum
	log.Printf("<Blog ID %d> 完成解析\n", id)
	dl.Img <- imgurls
}

// GetImgList 并行获取图片地址
func GetImgList(ym string, entries []string, blogInfo *BlogInfo) (imgList []string) {
	dl := new(DLserver)
	entryList := len(entries)
	dl.WG.Add(entryList)
	dl.Gonum = make(chan string, 8)
	dl.Img = make(chan []string)
	for id, e := range entries {
		dl.Gonum <- e
		go getImgURLEngine(id+1, e, dl, blogInfo)
	}
	dl.WG.Wait()
	imgList = <-dl.Img
	imgTotal := len(imgList)
	log.Printf("<%s> 有 %d 张图片(接口返回数据不准确)\n", ym, imgTotal)
	return
}

// downloadEngine 下载函数
func downloadEngine(id int, img string, path string, dl *DLserver) {
	para := make(map[string]string)
	urlCut := strings.Split(img, "/")
	filename := urlCut[len(urlCut)-1]
	savePath := path + "//" + filename
	imgExist, _ := pathExists(savePath)
	if !imgExist {
		log.Printf("<Downlaod ID %d> [%s Downloading...]\n", id, filename)
		res, _ := request("GET", img, nil, para, false)
		defer res.Body.Close()
		file, _ := os.Create(savePath)
		io.Copy(file, res.Body)
		log.Printf("<Downlaod ID %d> [%s is done]\n", id, filename)
	} else {
		log.Printf("<File ID %d> [%s Exists]\n", id, filename)
	}
	dl.WG.Done()
	<-dl.Gonum
}

// DownloadManger 图片下载管理
func DownloadManger(id int, imgurls []string, blogInfo *BlogInfo) {
	mainFolder := getCurrentDirectory() + "//" + blogInfo.author
	mainExist, _ := pathExists(mainFolder)
	if !mainExist {
		os.Mkdir(mainFolder, os.ModePerm)
	}
	dl := new(DLserver)
	runtime.GOMAXPROCS(runtime.NumCPU())
	tasks := len(imgurls)
	dl.WG.Add(tasks)
	log.Printf("<Master Task ID %d> 开始下载图片\n", id)
	dl.Gonum = make(chan string, 8)
	// 执行下载
	regDate := regexp.MustCompile(`2[0-9]{7}`)
	for mid, img := range imgurls {
		dl.Gonum <- img
		savename := regDate.FindAllString(img, -1)[0]
		subFolder := mainFolder + "//" + savename
		subExist, _ := pathExists(subFolder)
		if !subExist {
			os.Mkdir(subFolder, os.ModePerm)
		}
		go downloadEngine(mid+1, img, subFolder, dl)
	}
	dl.WG.Wait()
	log.Printf("<Master Task ID %d> 任务完成\n", id)
	fmt.Println("----------------------------")
}

// GetImgages 每抓取一个月执行下载
func GetImgages(uris []string, blogInfo *BlogInfo) {
	for id, ym := range uris {
		url := blogInfo.host + "/imagelist-" + ym + ".html"
		ymformat := ym[0:4] + "年" + ym[4:] + "月"
		blogInfo.target = ym
		firstEntries := GetFirstPageEntry(url)
		if len(firstEntries) == 0 {
			log.Printf("<%s> 无数据", ymformat)
		} else {
			nextEntries := GetOtherPageEntryByAPI(blogInfo)
			entryALL := append(firstEntries, nextEntries...)
			entryTotal := len(entryALL)
			log.Printf("<%s> 获取到 %d 篇博客\n", ymformat, entryTotal)
			imgurls := GetImgList(ymformat, entryALL, blogInfo)
			DownloadManger(id+1, imgurls, blogInfo)
		}
	}
}

func main() {
	// 接收地址
	tipURL := "A Blog URL\nHome Page - https://ameblo.jp/tao-tsuchiya"
	tipDate := "Date range 格式说明:\n\nYYYYMMM- 截止到当前年月\n-YYYYMM 从2005年到指定年月\nYYYYMM-YYYYMM 指定范围"
	fmt.Printf("---------Example-----------\n%s\n\n%s\n--------------------------\n", tipURL, tipDate)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("A Blog URL: ")
	data1, _, _ := reader.ReadLine()
	url, ok := urlCheck(string(data1))
	if ok {
		fmt.Print("Date range: ")
		data2, _, _ := reader.ReadLine()
		drange := string(data2)

		bloginfo := new(BlogInfo)
		bloginfo.author = strings.Split(url, "/")[3]
		bloginfo.host = url
		uris := TimeGenarater(drange)
		urisTotal := len(uris)
		if urisTotal > 0 {
			fmt.Printf("需要解析 %d 个月的数据...\n", urisTotal)
			GetImgages(uris, bloginfo)
		}
		fmt.Printf("所有任务完成\n\n")
		fmt.Println("Ctrl+C to exit.")
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
	} else {
		fmt.Println("Url is invalid ! Ctrl+C to exit.")
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
	}
}
