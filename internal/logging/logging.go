package logging

import (
	"log"

	"gopkg.in/natefinch/lumberjack.v2"
)

// update defaut log package's io writer to lumberjack's io writer
func SetupLogger(logDirPath string, logFileName string, logFileSizeMax int, logFilesAgeMax int, logFilesMax int, logCompress bool) {
	log.SetOutput(&lumberjack.Logger{
		Filename:   logDirPath + "/" + logFileName, //name of log file
		MaxSize:    logFileSizeMax,                 //maximum size of 1 log file
		MaxBackups: logFilesMax,                    //maximum nuber of log file the specified directory can have
		MaxAge:     logFilesAgeMax,                 //maximum age in days that a file can persist
		Compress:   logCompress,                    //should historical log files be compressed
	})
}
