package log

import (
	"log"
	"os"
)

func Debug(msg string, ctx ...interface{}) {
	log.Printf("[Debug]"+msg+"\n", ctx...)
}

func Info(msg string, ctx ...interface{}) {
	log.Printf("[Info]"+msg+"\n", ctx...)
}

func Warn(msg string, ctx ...interface{}) {
	log.Printf("[Warn]"+msg+"\n", ctx...)
}

func Error(msg string, ctx ...interface{}) {
	log.Printf("[Error]"+msg+"\n", ctx...)

}

func Fatal(msg string, ctx ...interface{}) {
	log.Printf("[Fatal]"+msg+"\n", ctx...)
	os.Exit(1)
}
