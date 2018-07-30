package log

import (
	"fmt"
	"os"
	"time"
)

func Debug(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[Debug]"+msg, ctx)
}

func Info(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[Info]"+msg, ctx)
}

func Warn(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[Warn]"+msg, ctx)
}

func Error(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[Error]"+msg, ctx)
}

func Fatal(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[Fatal]"+msg, ctx)
	os.Exit(1)
}
