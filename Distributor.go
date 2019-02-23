// test comment

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/a-palchikov/sqlago"
)

// MailDistributor ...
type MailDistributor struct {
	// connection string to database
	ConnectionString string
	// database driver name
	DBDriverName string
	// log filename
	LogFile string
	// Parameters for SMTP server
	SMTPParams struct {
		Host string
		Port int
	}
	// Parameters for distributing
	DistributeParams struct {
		LocalDomain   string
		LocalInterval int
		LocalCount    int
		OuterInterval int
		OuterCount    int
	}
	logger     *log.Logger
	loggerfile *os.File
	channel    chan string
	waitgrp    sync.WaitGroup
}

type mailMessage struct {
	sender      string
	recievers   string
	subject     string
	copyto      string
	body        string
	contenttype string
}

// Init - Load params from config.json and open log file ...
func (m *MailDistributor) Init() error {
	cfile, err := ioutil.ReadFile("config.json")
	if err != nil {
		m.printLog(false, true, "Configuration file not found!")
		return err
	}
	err = json.Unmarshal(cfile, &m)
	if err != nil {
		m.printLog(false, true, "Problem with config file")
		return err
	}

	m.loggerfile, err = os.OpenFile(m.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		m.printLog(false, true, "Failed to open log file")
		return err
	}

	m.logger = log.New(m.loggerfile, "", log.Ldate|log.Ltime)
	m.printLog(false, true, "Init (%s | %s)\n", m.ConnectionString, m.SMTPParams.Host)

	m.channel = make(chan string, 2)
	return nil
}

// Close - finish work, close log file
func (m *MailDistributor) Close() {
	m.waitgrp.Wait()
	m.printLog(false, true, "Stopped\n")
	m.loggerfile.Close()
}

func (email *mailMessage) buildMessage() string {
	message := ""
	message += fmt.Sprintf("From: %s\r\n", email.sender)
	message += fmt.Sprintf("To: %s\r\n", email.recievers)
	if email.copyto > "" {
		message += fmt.Sprintf("Cc: %s\r\n", email.copyto)
	}
	message += fmt.Sprintf("Content-Type: %s; charset=UTF-8\r\n", email.contenttype)
	message += fmt.Sprintf("Subject: %s\r\n", email.subject)
	message += "\r\n" + email.body

	return message
}

func (m *MailDistributor) prepareAndSendMessage(email *mailMessage) error {

	if strings.Contains(email.body, "<HTML") {
		email.contenttype = "text/html"
	} else {
		email.contenttype = "text/plain"
	}

	// Connect to the remote SMTP server.
	smtpclient, err := smtp.Dial(m.SMTPParams.Host + ":" + strconv.Itoa(m.SMTPParams.Port))
	if err != nil {
		return err
	}
	defer smtpclient.Close()
	err = smtpclient.Mail(email.sender)
	if err != nil {
		return err
	}

	for _, k := range strings.Split(email.recievers, ";") {
		err = smtpclient.Rcpt(k)
		if err != nil {
			return err
		}
	}

	for _, k := range strings.Split(email.copyto, ";") {
		err = smtpclient.Rcpt(k)
		if err != nil {
			return err
		}
	}

	// Send the email body.
	mbody, err := smtpclient.Data()
	if err != nil {
		return err
	}
	defer mbody.Close()

	messageBody := email.buildMessage()
	mbody.Write([]byte(messageBody))

	return nil
}

func (m *MailDistributor) printLog(fatal bool, both bool, format string, v ...interface{}) {
	if both {
		m.logger.Printf(format, v...)
		fmt.Printf(format, v...)
	} else {
		m.logger.Printf(format, v...)
	}
	if fatal {
		m.Close()
		os.Exit(1)
	}
}

// Start - start processing queue
func (m *MailDistributor) Start() {
	cmd := ""
	fmt.Printf("Started. Type \"exit\" to quit the program\n")

	m.waitgrp.Add(2)
	go m.ProcessQueue(&m.waitgrp, true)
	go m.ProcessQueue(&m.waitgrp, false)

	for {
		fmt.Scanln(&cmd)
		if cmd == "exit" {
			m.channel <- cmd
			m.channel <- cmd
			break
		}
	}
}

// ProcessQueue - send emails from queue, stored in db
func (m *MailDistributor) ProcessQueue(wg *sync.WaitGroup, islocal bool) {
	var cmd string
	defer wg.Done()

LOOP:
	for {

		select {
		case cmd = <-m.channel:
			if cmd == "exit" {
				if islocal {
					m.printLog(false, true, "Exiting(local)...\n")
				} else {
					m.printLog(false, true, "Exiting(outer)...\n")
				}
				break LOOP
			}
		default:
		}

		var condition string
		var mark string
		var db, err = sql.Open(m.DBDriverName, m.ConnectionString)
		if err != nil {
			m.printLog(true, true, "Unable to connect to db: %s\n", err)
		}

		if islocal {
			condition = ""
			mark = "(<->)"
		} else {
			condition = " not "
			mark = "(->)"
		}

		var rows, errq = db.Query("Select top " + strconv.Itoa(m.DistributeParams.LocalCount) +
			" l.LETTERID, l.SENDER,l.RECIEVER,l.CC,l.SUBJ,l.MSG from LETTERS_TO_SEND l where l.reciever " +
			condition + " like '%" + m.DistributeParams.LocalDomain + "%'" +
			" and not exists(select 1 from sended_letters s where s.letterid=l.letterid) order by l.LETTERID")

		if errq != nil {
			m.printLog(true, true, "Select from LETTERS_TO_SEND failed: %s\n", errq)
		}

		for rows.Next() {
			var email mailMessage
			var status string
			var letterid string

			rows.Scan(&letterid, &email.sender, &email.recievers, &email.copyto, &email.subject, &email.body)

			logstr := mark + "\n"
			logstr += fmt.Sprintf("\tMessage for %s\n", email.recievers)
			logstr += fmt.Sprintf("\tSubject: %s\n", email.subject)

			err := m.prepareAndSendMessage(&email)
			if err != nil {
				status = err.Error()
				logstr += fmt.Sprintf("\t" + status + "\n")
			} else {
				status = "ok"
				logstr += fmt.Sprintf("\tSended from: %s\n\n", email.sender)
			}

			m.printLog(false, false, logstr)

			db.Exec("insert into SENDED_LETTERS (LetterID,Date_send,ErrorMessage) values('" + letterid + "',getdate(),'" + status + "')")
		}

		if islocal {
			time.Sleep(time.Duration(m.DistributeParams.LocalInterval) * time.Second)
		} else {
			time.Sleep(time.Duration(m.DistributeParams.OuterInterval) * time.Second)
		}
	}
}
