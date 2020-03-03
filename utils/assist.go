package utils

import (
    "io/ioutil"
    "log"
    "os"
    "reflect"
    "strconv"
    "strings"
)

// CommonUA 全局 UA
func CommonUA() string {
    userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.122 Safari/537.36"
    return userAgent
}

// CheckType 检查类型
func CheckType(i interface{}) reflect.Type {
    return reflect.TypeOf(i)
}

// GetCurrentDirectory 获取当前路径
func GetCurrentDirectory() string {
    dir, _ := os.Getwd()
    return strings.Replace(dir, "\\", "/", -1)
}

// PathExists 判断文件夹是否存在
func PathExists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil {
        return true, nil
    }
    if os.IsNotExist(err) {
        return false, nil
    }
    return false, err
}

// RemoveDuplicate 去重
func RemoveDuplicate(s []string) (ret []string) {
    LEN := len(s)
    for i := 0; i < LEN; i++ {
        if (i > 0 && s[i-1] == s[i]) || len(s[i]) == 0 {
            continue
        }
        ret = append(ret, s[i])
    }
    return
}

// Int2strAdd0 数字转字符串并补零
func Int2strAdd0(i int) (s string) {
    S := strconv.Itoa(i)
    if len(S) == 1 {
        s = "0" + S
    } else {
        s = S
    }
    return
}

// Str2Int 数字转字符串并补零
func Str2Int(s string) (i int) {
    result, err := strconv.Atoi(s)
    if err != nil {
        log.Fatal(err)
    }
    i = result
    return i
}

// OpenFile 打开文件
func OpenFile(s string) (b []byte) {
    data, err := ioutil.ReadFile(s)
    if err != nil {
        log.Fatal(err)
    }
    b = data
    return
}

// WriteFile 写文件
func WriteFile(s string, b []byte) {
    ioutil.WriteFile(s, b, 0666)
}