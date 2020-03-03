package main

import (
    "amebloimg-go/ameblo"
    "bufio"
    "fmt"
    "log"
    "os"
    "os/signal"
    "strings"
)

// urlCheck 检查地址有效性
func urlCheck(url string) (newurl string, b bool) {
    newurl = strings.TrimSpace(url)
    if strings.Contains(newurl, "https://ameblo.jp/") {
        b = true
    } else {
        b = false
    }
    return
}

func main() {
    // 接收地址
    tipURL := "A Blog URL\nHome Page - https://ameblo.jp/tao-tsuchiya"
    tipDate := "Date range 格式说明:\n\nYYYYMMM- 截止到当前年月\n-YYYYMM 从2005年到指定年月\nYYYYMM-YYYYMM 指定范围"
    fmt.Printf("---------Example-----------\n%s\n\n%s\n--------------------------\n", tipURL, tipDate)

    reader := bufio.NewReader(os.Stdin)
    fmt.Print("A Blog URL: ")
    homeURL, _, _ := reader.ReadLine()

    url, ok := urlCheck(string(homeURL))
    if ok {
        fmt.Print("Date range: ")
        dateRange, _, _ := reader.ReadLine()

        drange := strings.TrimSpace(string(dateRange))
        uris := ameblo.TimeGenarater(drange)
        urisTotal := len(uris)
        if urisTotal > 0 {
            fmt.Printf("需要解析 %d 个月的数据...\n", urisTotal)
            for i := 0; i < urisTotal; i++ {
                period := uris[i]
                yearMonth := period[0:4] + "年" + period[4:] + "月"
                author, amebloEIDs := ameblo.GetAllEntryIDs(url, period)
                entryTotal := len(amebloEIDs)
                if entryTotal == 0 {
                    log.Printf("<%s> 无数据", yearMonth)
                } else {
                    log.Printf("获取到 %s 的 %d 篇博客\n", yearMonth, entryTotal)
                    log.Printf("开始识别 %s 的所有图片地址\n", yearMonth)
                    imgURLs := ameblo.GetImgURLs(author, period, amebloEIDs)
                    imgTotal := len(imgURLs)
                    log.Printf("已解析到 %s 的 %d 图片\n", yearMonth, imgTotal)
                    ameblo.DownloadImg(author, period, imgURLs)
                    log.Printf("完成 %s 的任务\n", yearMonth)
                }
            }
            log.Println("所有任务完成")
            fmt.Println("Ctrl+C to exit.")
            c := make(chan os.Signal, 1)
            signal.Notify(c, os.Interrupt, os.Kill)
            <-c
        }
    } else {
        fmt.Println("Url is invalid ! Ctrl+C to exit.")
        c := make(chan os.Signal, 1)
        signal.Notify(c, os.Interrupt, os.Kill)
        <-c
    }
}
