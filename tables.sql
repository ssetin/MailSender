CREATE TABLE LETTERS_TO_SEND (LETTERID varchar(20) PRIMARY KEY,SENDER varchar(60),RECIEVER varchar(60),CC varchar(60),SUBJ varchar(128),MSG long varchar)

CREATE TABLE SENDED_LETTERS (LETTERID varchar(20) PRIMARY KEY, DATE_SEND datetime, ERRORMESSAGE varchar(128))