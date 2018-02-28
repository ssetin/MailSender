# Mailsender - tool to distribute emails

Mailsender can be used in your reports or notification subsystem as a email queue processing tool. It helps to decrease loading on smtp server in case of plenty messages. To send email you should add new record to table LETTER_TO_SEND in your database. Mailsender checks for new messages every specified period of time and send them by small portions.

## Deploy

To deploy that tool you should create tables from tables.sql in your database

## Configuring

Configuration parameters are stored in config.json.

```json
{
	"ConnectionString": "commlinks=tcpip;uid=dba;pwd=SQL;ServerName=server;DatabaseName=database",
	"DBDriverName": "sqlany",
	"LogFile": "logme.log",
	"SMTPParams": { "Host": "mail.spb.avantel.ru", "Port": 25},
	"DistributeParams": {"LocalDomain":"@domain.com", "LocalInterval": 30,"LocalCount": 5, "OuterInterval": 60,"OuterCount": 5 }
}
```
#### DistributeParams
- LocalDomain - the domain for which the local distribution rules will be used
- LocalInterval - sending period for local messages
- LocalCount - the number of sent messages for local domain
- OuterInterval - sending period for non local messages
- OuterCount - the number of sent messages for non local domain