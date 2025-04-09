package utils

import (
	"math"
	"math/rand"
	"net"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/google/uuid"
)

// ConcatStrings 将多个字符串合成一个字符串数组
func ConcatStrings(elems ...string) []string {
	return elems
}

// StrSplit 使用逗号或者分号对字符串进行切割
func StrSplit(s string, p ...rune) []string {
	return strings.FieldsFunc(s, func(c rune) bool { return c == ',' || c == ';' })
}

// StringToBytes 直接将字符串转换成[]byte，无内存copy
func StringToBytes(s string) []byte {
	stringHeader := unsafe.StringData(s)
	return unsafe.Slice(stringHeader, len(s))
}

// BytesToString 直接将[]byte转成字符串，无内存copy
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// CompressStr 压缩字符串，去掉所有空白符
func CompressStr(str string) string {
	if str == "" {
		return ""
	}
	//匹配一个或多个空白符的正则表达式
	reg := regexp.MustCompile("\\s+")
	return reg.ReplaceAllString(str, "")
}

func StringToInt32(numStr string) int32 {
	num, _ := strconv.Atoi(numStr)
	return int32(num)
}

func StringToFloat64(num string) float64 {
	fnum, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0
	}
	return fnum
}

func Float64ToString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func FormatFloat(value float64, precision float64) float64 {
	precision = math.Pow(10, precision)
	return math.Round(value*precision) / precision
}

// 获取标准的uuid字符串
func GetUUID() string {
	return uuid.New().String()
}

// 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// 将int64毫秒级时间戳转换为"2006-01-02 15:04:05.000"GTC时间格式
func Int64ToTimeFormat(timestamp int64) string {
	sec := timestamp / 1000
	nsec := (timestamp % 1000) * int64(time.Millisecond)
	t := time.Unix(sec, nsec)
	return t.Format("2006-01-02 15:04:05.000")
}

// 将int64毫秒级时间间隔转换为时分秒的格式
func Int64ToTimeDuration(timestamp int64) string {
	d := time.Duration(timestamp) * time.Millisecond
	// 提取小时、分钟和秒
	hours := d / time.Hour
	d %= time.Hour
	minutes := d / time.Minute
	d %= time.Minute
	seconds := d / time.Second

	return strconv.Itoa(int(hours)) + "时" + strconv.Itoa(int(minutes)) + "分" + strconv.Itoa(int(seconds)) + "秒"
}

// 获取本机所有ip
func GetAllLocalIPs() ([]string, error) {
	ipArr := make([]string, 0)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ipArr, err
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipArr = append(ipArr, ipnet.IP.String())
			}
		}
	}
	return ipArr, nil
}

// LineInfo returns the function name, file name and line number of the caller function.
func LineInfo() string {
	function := "xxx"
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	function = runtime.FuncForPC(pc).Name()

	return strings.Join(ConcatStrings(file, "(", function, "):", strconv.Itoa(line)), "")
}
