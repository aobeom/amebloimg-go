package ameblo

import (
    "amebloimg-go/utils"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "runtime"
    "sort"
    "strconv"
    "strings"
    "sync"
    "time"
)

// DLserver 并行控制
type DLserver struct {
    WG    sync.WaitGroup
    Gonum chan string
    Data  []string
}

// BlogInfo 博客基本信息
type BlogInfo struct {
    Author  string
    Target  string
    Host    string
    EntryID string
}

// timeFormat 转换时间格式
func timeFormat(s string) string {
    t, _ := time.Parse("2006/01/02", s)
    return t.Format("200601")
}

// makeUserInfo 获取 URL 信息
func makeUserInfo(url string) (host, author string) {
    urlData := strings.Split(url, "/")
    host = strings.TrimRight(url, "/")
    author = urlData[3]
    return
}

// filterJSON 匹配 JSON 数据
func filterJSON(b []byte) (data map[string]interface{}) {
    blogRule := regexp.MustCompile(`<script>window.INIT_DATA=(.*?);window.RESOURCE_BASE_URL`)
    blogRawData := blogRule.FindAllStringSubmatch(string(b), -1)
    if len(blogRawData) != 0 {
        jsonStringData := strings.Replace(blogRawData[0][1], "\\u002F", "/", -1)
        json.Unmarshal([]byte(jsonStringData), &data)
        // utils.WriteFile("4.json", []byte(jsonStringData))
    }
    return
}

// makeJSON 生成 JSON 数据
func makeJSON(b []byte) (data map[string]interface{}) {
    json.Unmarshal(b, &data)
    return
}

// EntryOfFirst 获取博客 JSON 数据
func entryOfFirst(para map[string]string) (eIDs []string, nextURL string) {
    author := para["author"]
    year := para["year"]
    month := para["month"]
    url := para["url"]
    headers := make(http.Header)
    headers.Add("User-Agent", utils.CommonUA())
    resBody := utils.Get(url, headers, nil)

    firstData := filterJSON(resBody)

    bImageState := firstData["imageState"].(map[string]interface{})
    bImageArchiveMap := bImageState["imageArchiveMap"].(map[string]interface{})
    bImageMetaMap := bImageState["imageMetaMap"].(map[string]interface{})
    bFirstEntryID := bImageMetaMap["firstEntryId"]

    if bFirstEntryID != nil {
        bAuthor := bImageArchiveMap[author].(map[string]interface{})
        bYear := bAuthor[year].(map[string]interface{})
        bMonth := bYear[month].(map[string]interface{})
        bImageData := bMonth["imageData"].([]interface{})
        nextURL = bMonth["nextUrl"].(string)
        for i := 0; i < len(bImageData); i++ {
            iData := bImageData[i].(map[string]interface{})
            iDate := timeFormat(iData["date"].(string))
            if iDate == month {
                iEntryID := strconv.FormatFloat(iData["entryId"].(float64), 'f', -1, 64)
                eIDs = append(eIDs, iEntryID)
            } else {
                nextURL = ""
            }
        }
        eIDs = utils.RemoveDuplicate(eIDs)
    } else {
        eIDs = []string{}
        nextURL = ""
    }
    return
}

func entryOfOthers(url, period string) (eIDs []string, nextURL string) {
    headers := make(http.Header)
    headers.Add("User-Agent", utils.CommonUA())
    resBody := utils.Get(url, headers, nil)
    otherData := makeJSON(resBody)
    bData := otherData["data"].([]interface{})
    bPaging := otherData["paging"].(map[string]interface{})
    nextURL = bPaging["nextUrl"].(string)
    for i := 0; i < len(bData); i++ {
        iData := bData[i].(map[string]interface{})
        iDate := timeFormat(iData["date"].(string))
        if iDate == period {
            iEntryID := strconv.FormatFloat(iData["entryId"].(float64), 'f', -1, 64)
            eIDs = append(eIDs, iEntryID)
        } else {
            nextURL = ""
        }
    }
    return
}

// GetAllEntryIDs 获取所有博客 ID
func GetAllEntryIDs(url, period string) (author string, allIDs []string) {
    host, author := makeUserInfo(url)
    imgURL := host + "/imagelist-" + period + ".html"
    entryPara := make(map[string]string)
    entryPara["url"] = imgURL
    entryPara["author"] = author
    entryPara["year"] = period[:4]
    entryPara["month"] = period

    // data, _ := ioutil.ReadFile("1.html")
    // getJSONString(data)
    // fmt.Println("")

    firstEntries, nextURL := entryOfFirst(entryPara)
    allIDs = append(allIDs, firstEntries...)
    if len(firstEntries) == 0 {
        allIDs = []string{}
    } else {
        if nextURL != "" {
            nextEntries, nextURL2 := entryOfOthers(nextURL, period)
            allIDs = append(allIDs, nextEntries...)
            for nextURL2 != "" {
                nextEntries, nextURL2 = entryOfOthers(nextURL2, period)
                allIDs = append(allIDs, nextEntries...)
            }
        }
    }
    return
}

func getImgEngine(dl *DLserver, author, period string, entryID string) {
    imgHost := "http://stat.ameba.jp"
    imgAPIURL := "https://blogimgapi.ameba.jp/read_ahead/get.json"

    headers := make(http.Header)
    headers.Add("User-Agent", utils.CommonUA())

    params := map[string]string{
        "ameba_id": author,
        "entry_id": entryID,
        "old":      "false",
        "sp":       "true",
    }
    log.Printf("<Entry ID %s> 解析图片地址 \n", entryID)
    resBody := utils.Get(imgAPIURL, headers, params)
    bImgData := makeJSON(resBody)
    bImgList := bImgData["imgList"].([]interface{})
    for i := 0; i < len(bImgList); i++ {
        iData := bImgList[i].(map[string]interface{})
        iDate := timeFormat(iData["date"].(string))
        if iDate == period {
            iImgURL := imgHost + iData["imgUrl"].(string) + "?caw"
            dl.Data = append(dl.Data, iImgURL)
        }
    }
    dl.WG.Done()
    log.Printf("<Entry ID %s> 完成解析 \n", entryID)
    <-dl.Gonum
}

// GetImgURLs 获取所有图片地址
func GetImgURLs(author, period string, entryIDs []string) (imgURLs []string) {
    dl := new(DLserver)
    entryList := len(entryIDs)
    dl.WG.Add(entryList)
    dl.Gonum = make(chan string, 8)

    for i := 0; i < len(entryIDs); i++ {
        entryID := entryIDs[i]
        dl.Gonum <- entryID
        go getImgEngine(dl, author, period, entryID)
    }
    dl.WG.Wait()
    sort.Strings(dl.Data)
    imgURLs = utils.RemoveDuplicate(dl.Data)
    return
}

func makeFolder(s string) (f string) {
    f = filepath.Join(utils.GetCurrentDirectory(), s)
    fExist, _ := utils.PathExists(f)
    if !fExist {
        os.Mkdir(f, os.ModePerm)
    }
    return
}

func downloadEngine(dl *DLserver, taskID string, imgURL string, saveFolder string) {
    imgParaDel := strings.Split(imgURL, "?")[0]
    imgURLSplit := strings.Split(imgParaDel, "/")
    imgName := imgURLSplit[len(imgURLSplit)-1]
    savePath := filepath.Join(saveFolder, imgName)
    imgExist, _ := utils.PathExists(savePath)

    headers := make(http.Header)
    headers.Add("User-Agent", utils.CommonUA())

    if !imgExist {
        log.Printf("<IMG ID %s> Processing...", taskID)
        resBody := utils.Get(imgURL, headers, nil)
        bFile, _ := os.Create(savePath)
        resReader := strings.NewReader(string(resBody))
        io.Copy(bFile, resReader)
        defer bFile.Close()
        log.Printf("<IMG ID %s> Finished.", taskID)
    } else {
        log.Printf("<IMG ID %s> %s Exists\n", taskID, imgName)
    }
    dl.WG.Done()
    <-dl.Gonum
}

// DownloadImg 下载所有图片
func DownloadImg(author, period string, imgURLs []string) {
    mainFolder := makeFolder(author)
    dl := new(DLserver)
    runtime.GOMAXPROCS(runtime.NumCPU())
    imgTasks := len(imgURLs)
    dl.WG.Add(imgTasks)
    dl.Gonum = make(chan string, 8)

    log.Printf("<Task ID %s> 开始下载\n", period)
    for imgID, imgURL := range imgURLs {
        taskID := period + "-" + strconv.Itoa(imgID+1)
        dl.Gonum <- imgURL
        if strings.Contains(imgURL, "filtering_ng") {
            log.Printf("<Task ID %s> 图片失效 %s\n", period, imgURL)
        } else {
            saveFolder := filepath.Join(mainFolder, period)
            saveFolderExist, _ := utils.PathExists(saveFolder)
            if !saveFolderExist {
                os.Mkdir(saveFolder, os.ModePerm)
            }
            go downloadEngine(dl, taskID, imgURL, saveFolder)
        }
    }
    dl.WG.Wait()
    log.Printf("<Task ID %s> 下载完成\n", period)
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
    today := time.Now().Format("200601")
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
                yearMAX := utils.Str2Int(year)
                monthMAX := utils.Str2Int(month)
                // 开始时间统一是amblo创立时间
                for yearMIN := 2005; yearMIN <= yearMAX; yearMIN++ {
                    var newD string
                    // 截止到指定的年月
                    if yearMIN == yearMAX {
                        for monthMIN := 1; monthMIN <= monthMAX; monthMIN++ {
                            newD = utils.Int2strAdd0(yearMIN) + utils.Int2strAdd0(monthMIN)
                            DATE = append(DATE, newD)
                        }
                        // 中间年份自动填充12个月
                    } else {
                        for monthMIN := 1; monthMIN <= 12; monthMIN++ {
                            newD = utils.Int2strAdd0(yearMIN) + utils.Int2strAdd0(monthMIN)
                            DATE = append(DATE, newD)
                        }
                    }
                }
            } else {
                log.Fatal("Time Error")
            }
            // 只有开始时间
        } else if datePart[1] == "" {
            d, ok := timeCheck(datePart[0])
            if ok {
                // 当前年月
                yearNow := today[0:4]
                monthNow := today[4:]
                yearMAX := utils.Str2Int(yearNow)
                monthMAX := utils.Str2Int(monthNow)
                // 指定年月
                year := d[0:4]
                month := d[4:]
                yearMIN := utils.Str2Int(year)
                monthMIN := utils.Str2Int(month)
                // 第一年从指定月份开始
                if yearMIN != yearMAX {
                    for monthMIN <= 12 {
                        var newD string
                        newD = utils.Int2strAdd0(yearMIN) + utils.Int2strAdd0(monthMIN)
                        DATE = append(DATE, newD)
                        monthMIN++
                    }
                } else {
                    for monthMIN <= monthMAX {
                        var newD string
                        newD = utils.Int2strAdd0(yearMIN) + utils.Int2strAdd0(monthMIN)
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
                            newD = utils.Int2strAdd0(yearMINnext) + utils.Int2strAdd0(monthMIN)
                            DATE = append(DATE, newD)
                        }
                        // 中间年份自动填充12个月
                    } else {
                        for monthMIN := 1; monthMIN <= 12; monthMIN++ {
                            newD = utils.Int2strAdd0(yearMINnext) + utils.Int2strAdd0(monthMIN)
                            DATE = append(DATE, newD)
                        }
                    }
                    yearMINnext++
                }
            } else {
                log.Fatal("Time Error")
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
                    yearMIN := utils.Str2Int(yearSta)
                    monthMIN := utils.Str2Int(monthSta)
                    yearMAX := utils.Str2Int(yearEnd)
                    monthMAX := utils.Str2Int(monthEnd)
                    // 第一年从指定月开始
                    if yearMIN != yearMAX {
                        for monthMIN <= 12 {
                            var newD string
                            newD = utils.Int2strAdd0(yearMIN) + utils.Int2strAdd0(monthMIN)
                            DATE = append(DATE, newD)
                            monthMIN++
                        }
                    } else {
                        for monthMIN <= monthMAX {
                            var newD string
                            newD = utils.Int2strAdd0(yearMIN) + utils.Int2strAdd0(monthMIN)
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
                                newD = utils.Int2strAdd0(yearMINnext) + utils.Int2strAdd0(monthMIN)
                                DATE = append(DATE, newD)
                            }
                            // 中间年份自动填充12个月
                        } else {
                            for monthMIN := 1; monthMIN <= 12; monthMIN++ {
                                newD = utils.Int2strAdd0(yearMINnext) + utils.Int2strAdd0(monthMIN)
                                DATE = append(DATE, newD)
                            }
                        }
                        yearMINnext++
                    }
                } else {
                    log.Fatal("Time Error")
                }
            } else {
                log.Fatal("Time Error")
            }
        }
    } else if checkSep == 1 {
        d, ok := timeCheck(datePart[0])
        if ok {
            DATE = append(DATE, d)
        } else {
            log.Fatal("Time Error")
        }
    } else {
        log.Fatal("Time Error")
    }
    return
}
