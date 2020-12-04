package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"wataru.com/gogo/util"
)

const (
	PanicLevel int = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
)

const (
	ByDay int = iota
	ByWeek
	ByMonth
	BySize
)

const (
	color_red     = uint8(iota + 91)
	color_green   //    绿
	color_yellow  //    黄
	color_blue    //     蓝
	color_magenta //    洋红
)

const (
	fatalPrefix = "[FATAL] "
	errorPrefix = "[ERROR] "
	warnPrefix  = "[WARN]  "
	infoPrefix  = "[INFO]  "
	debugPrefix = "[DEBUG] "
)

type LogFile struct {
	level    int    // 日志等级
	saveMode int    // 保存模式
	saveDays int    // 日志保存天数
	logTime  int64  //
	fileName string // 日志文件名
	filesize int64  // 文件大小, 需要设置 saveMode 为 BySize 生效
	fileFd   *os.File
}

var logFile LogFile
var logWritter io.Writer
var logger *log.Logger
var gormLogger *GormLogger
var rawLogger *log.Logger

func (l *LogFile) formatMsgHeader(calldepth int, prefix string, codeLine string) string {
	now := time.Now() // get this early.
	var file string
	// Release lock while getting caller info - it's expensive.
	var ok bool
	var line int
	if codeLine != "" {
		file = codeLine
	} else if _, file, line, ok = runtime.Caller(calldepth); ok {
		file = file + ":" + strconv.Itoa(line)
	} else {
		file = "???"
	}
	goroutine := ""
	buf := []byte{}
	l.formatHeader(&buf, prefix, now, file, goroutine)
	return string(buf)
}

func (l *LogFile) formatHeader(buf *[]byte, prefix string, t time.Time, file string, goroutine string) {
	*buf = append(*buf, prefix...)
	// year, month, day := t.Date()
	// itoa(buf, year, 4)
	// *buf = append(*buf, '-')
	// itoa(buf, int(month), 2)
	// *buf = append(*buf, '-')
	// itoa(buf, day, 2)
	// *buf = append(*buf, ' ')
	hour, min, sec := t.Clock()
	itoa(buf, hour, 2)
	*buf = append(*buf, ':')
	itoa(buf, min, 2)
	*buf = append(*buf, ':')
	itoa(buf, sec, 2)
	*buf = append(*buf, '.')
	itoa(buf, t.Nanosecond()/1e3, 6)
	*buf = append(*buf, ' ')
	f := fmt.Sprintf("%30s", file)
	*buf = append(*buf, ("[" + f[len(f)-30:] + "]")...)
	if goroutine != "" {
		*buf = append(*buf, fmt.Sprintf(" [%-12s]", goroutine)...)
	}
	*buf = append(*buf, ": "...)
}

// Cheap integer to fixed-width decimal ASCII. Give a negative width to avoid zero-padding.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

func Debug(format string, v ...interface{}) {
	if logFile.level >= DebugLevel {
		header := logFile.formatMsgHeader(2, plain(debugPrefix), "")
		_ = logger.Output(2, NewMessageString(header+fmt.Sprintf(format, v...)))
	}
}

func Info(format string, v ...interface{}) {
	if logFile.level >= InfoLevel {
		header := logFile.formatMsgHeader(2, plain(infoPrefix), "")
		_ = logger.Output(2, NewMessageString(header+fmt.Sprintf(format, v...)))
	}
}

func logSql(format string, v ...interface{}) {
	if logFile.level >= InfoLevel {
		codeLine, _ := v[1].(string)
		params, _ := v[4].([]interface{})
		var formatParams []interface{}
		for _, item := range params {
			if v1, ok1 := item.(*string); ok1 && v1 != nil {
				formatParams = append(formatParams, *v1)
			} else if v2, ok2 := item.(*int); ok2 && v2 != nil {
				formatParams = append(formatParams, *v2)
			} else if v3, ok3 := item.(*int64); ok3 && v3 != nil {
				formatParams = append(formatParams, *v3)
			} else {
				formatParams = append(formatParams, item)
			}
		}
		header := logFile.formatMsgHeader(2, plain(infoPrefix), codeLine)
		_ = logger.Output(2, NewMessageString(header+fmt.Sprintf(format, v[0], v[3], formatParams, v[2])))
	}
}

type GormLogger struct {
}

// v[0] level
// v[1] file
// v[2] time
// v[3] sql
func (l *GormLogger) Print(v ...interface{}) {
	logSql("[%s] %s %v %s", v...)
}

func Warn(format string, v ...interface{}) {
	if logFile.level >= WarnLevel {
		header := logFile.formatMsgHeader(2, plain(warnPrefix), "")
		_ = logger.Output(2, NewMessageString(header+fmt.Sprintf(format, v...)))
	}
}

func Error(format string, v ...interface{}) {
	if logFile.level >= ErrorLevel {
		header := logFile.formatMsgHeader(2, plain(errorPrefix), "")
		_ = logger.Output(2, NewMessageString(header+fmt.Sprintf(format, v...)))
	}
}

func Fatal(format string, v ...interface{}) {
	if logFile.level >= FatalLevel {
		header := logFile.formatMsgHeader(2, plain(fatalPrefix), "")
		_ = logger.Output(2, NewMessageString(header+fmt.Sprintf(format, v...)))
	}
}

func Raw(format string, v ...interface{}) {
	if logFile.level >= InfoLevel {
		_ = rawLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func NewMessageString(message string) string {
	return message
}

func (me *LogFile) createLogFile() {
	if index := strings.LastIndex(me.fileName, "/"); index != -1 {
		_ = os.MkdirAll(me.fileName[0:index], os.ModePerm)
	}
	now := time.Now()
	filename := fmt.Sprintf("%s_%04d%02d%02d.log", me.fileName, now.Year(), now.Month(), now.Day())
	me.fileFd = nil
	if fd, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666); nil == err {
		_ = me.fileFd.Sync()
		_ = me.fileFd.Close()
		me.fileFd = fd
	} else {
		fmt.Println("Open logfile error! err: ", err.Error())
	}
}

func (me LogFile) Write(buf []byte) (n int, err error) {
	if me.fileName == "" {
		fmt.Printf("console: %s", buf)
		return len(buf), nil
	}
	now := time.Now()
	switch logFile.saveMode {
	case BySize:
		fileInfo, err := os.Stat(logFile.fileName)
		if err != nil {
			logFile.createLogFile()
			logFile.logTime = now.Unix()
		} else {
			filesize := fileInfo.Size()
			if logFile.fileFd == nil ||
				filesize > logFile.filesize {
				logFile.createLogFile()
				logFile.logTime = now.Unix()
			}
		}
	default: // 默认按天  ByDay
		if logFile.logTime+86400 < now.Unix() || logFile.logTime > now.Unix() {
			logFile.createLogFile()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
			logFile.logTime = today.Unix()
		}
	}

	if logFile.fileFd == nil {
		return len(buf), nil
	}

	return logFile.fileFd.Write(buf)
}

func Config(logConf map[interface{}]interface{}, level int, saveMode int, saveDays int) {
	logFolder := util.ValueOrDefault(logConf["path"], "log/app").(string)
	logFile.fileName = logFolder
	logFile.level = level
	logFile.saveMode = saveMode
	logFile.saveDays = saveDays
	logWritter = io.MultiWriter(os.Stdout, logFile)
	logger = log.New(logWritter, "", 0)
	gormLogger = &GormLogger{}
	rawLogger = log.New(logWritter, "", 0)
}

func GetLogFile() *LogFile {
	return &logFile
}

func GetWritter() *io.Writer {
	return &logWritter
}

func GetLogger() *log.Logger {
	return logger
}

func GetGormLogger() *GormLogger {
	return gormLogger
}

func SetLevel(level int) {
	logFile.level = level
}

func SetSaveMode(saveMode int) {
	logFile.saveMode = saveMode
}

func SetSaveDays(saveDays int) {
	logFile.saveDays = saveDays
}

func SetSaveSize(saveSize int64) {
	logFile.filesize = saveSize
}

func red(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_red, s)
}

func green(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_green, s)
}

func yellow(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_yellow, s)
}

func blue(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_blue, s)
}

func magenta(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color_magenta, s)
}

func plain(s string) string {
	return s
}
