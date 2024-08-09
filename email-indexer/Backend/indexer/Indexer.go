package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"indexer/zincsearch"
	_ "net/http/pprof"
	"net/mail"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
)

const (
	dbDirectory               = "./enron_mail_20110402"
	maxSizeEmail              = 1000000
	maxRoutinesToProcessEmail = 20
	capacityOfChannelZinc     = 20
	capacityOfChannelEmail    = 500
	capacityOfChannelInfo     = 500
	batchEmail                = 50
	layout                    = "Mon, 02 Jan 2006 15:04:05 -0700 (MST)"
	layout2                   = "Mon, 2 Jan 2006 15:04:05 -0700 (MST)"
)

var numRoutinesToProcessEmail = make(chan struct{}, maxRoutinesToProcessEmail)
var numRoutinesUploadEmails = make(chan struct{}, capacityOfChannelZinc)
var emailsDirectory = make(chan string, capacityOfChannelEmail)
var infoEmails = make(chan string, capacityOfChannelInfo)
var countEmailInserted, countEmailRejected, countEmailWrongFormat, countEmailsBig int
var waitProcessEmail, waitEmailResume sync.WaitGroup
var mutex sync.Mutex
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	_ = zinc.DeleteIndex()
	err := zinc.CreateIndex()
	if err != nil {
		panic(err)
	}
	timeBegin := time.Now()

	//Create a resume with all relevant info from emails
	waitEmailResume.Add(1)
	go createEmailResume(infoEmails)

	//Process emails from directory and get relevant info
	go routineToProcessEmails(emailsDirectory)

	getEmailsDirectory(dbDirectory)
	close(emailsDirectory)
	waitProcessEmail.Wait()
	close(infoEmails)
	waitEmailResume.Wait()

	fmt.Printf("Inserted:%d\n", countEmailInserted)
	fmt.Printf("Error formato:%d\n", countEmailWrongFormat)
	fmt.Printf("Rechazados batch:%d\n", countEmailRejected)
	fmt.Printf("Email to big:%d\n", countEmailsBig)
	fmt.Printf("\"\": %v\n", "")
	fmt.Printf("Duracion:%s\n", time.Since(timeBegin))
	fmt.Printf("\"\": %v\n", "")

	fmt.Println("End of process")

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}

// Get the directory of the database and get the
func getEmailsDirectory(rootDirectory string) {
	directory, err := os.Open(rootDirectory)
	if err != nil {
		panic(err.Error())
	}
	//Read the files in root
	files, err := directory.ReadDir(-1)
	if err != nil {
		panic(err.Error())
	}
	_ = directory.Close()
	for _, file := range files {
		if file.IsDir() {
			//Recursive to get files in the folder/directory
			getEmailsDirectory(rootDirectory + "/" + file.Name())
		} else {
			//Get info from file
			fileInfo, err := file.Info()
			if err != nil {
				panic(err.Error())
			}
			// If file is to big is not added
			if fileInfo.Size() > maxSizeEmail {
				countEmailsBig++
				continue
			}
			waitProcessEmail.Add(1)
			emailsDirectory <- rootDirectory + "/" + file.Name()
		}
	}
}

func routineToProcessEmails(emailsDirectory chan string) {
	for emailDir := range emailsDirectory {
		go processEmail(emailDir)
	}
}

func processEmail(email string) {
	defer func() {
		<-numRoutinesToProcessEmail
		waitProcessEmail.Done()
	}()

	numRoutinesToProcessEmail <- struct{}{}

	fileContent, err := os.ReadFile(email)
	if err != nil {
		panic(err.Error())
	}
	reader := bytes.NewReader(fileContent)
	message, err := mail.ReadMessage(reader)
	if err != nil {
		emailWrongFormat(email)
		return
	}
	body, err := io.ReadAll(message.Body)
	if err != nil {
		emailWrongFormat(email)
		return
	}

	date, err := time.Parse(layout, message.Header.Get("Date"))
	if err != nil {
		date, _ = time.Parse(layout2, message.Header.Get("Date"))
	}
	formatDate := date.Format(time.RFC3339)

	infoEmails <- fmt.Sprintf(`{"_id": "%s", "directory": "%s",  "from": %s, "to": %s, "subject": %s, "content": %s, "date": "%s"}`,
		message.Header.Get("Message-ID"), email, fmt.Sprintf("%q", message.Header.Get("From")), fmt.Sprintf("%q", message.Header.Get("To")),
		fmt.Sprintf("%q", message.Header.Get("Subject")), fmt.Sprintf("%q", strings.ReplaceAll(string(body), "\"", "'")), formatDate)

}

func emailWrongFormat(email string) {
	mutex.Lock()
	countEmailWrongFormat++
	mutex.Unlock()
}

func createEmailResume(infoEmails chan string) {
	defer waitEmailResume.Done()
	emailsResume := strings.Builder{}
	numberOfEmails := 0
	for email := range infoEmails {
		if emailsResume.Len()+len(email) > maxSizeEmail || numberOfEmails == batchEmail {
			waitEmailResume.Add(1)
			go uploadEmails(emailsResume.String(), numberOfEmails)
			numberOfEmails = 0
			emailsResume.Reset()
		}
		emailsResume.WriteString(email)
		emailsResume.WriteByte(10)
		numberOfEmails++
	}
	if emailsResume.Len() != 0 {
		waitEmailResume.Add(1)
		go uploadEmails(emailsResume.String(), numberOfEmails)
		numberOfEmails = 0
		emailsResume.Reset()
	}
}

func uploadEmails(emailsResume string, numberOfEmail int) {
	defer waitEmailResume.Done()
	numRoutinesUploadEmails <- struct{}{}
	count, err := zinc.CreateData(emailsResume)
	if err != nil {
		panic(err)
	}
	mutex.Lock()
	countEmailInserted += count
	countEmailRejected += numberOfEmail - count
	mutex.Unlock()
	<-numRoutinesUploadEmails
}
