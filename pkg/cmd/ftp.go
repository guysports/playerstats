package cmd

import (
	"crypto/tls"
	"fmt"

	"github.com/jlaffaye/ftp"
)

type FTP struct{}

func (f *FTP) Run(globals *Globals) error {
	ftpClient, err := ftp.Dial("ftp.guysports.co.uk:21", ftp.DialWithExplicitTLS(&tls.Config{ServerName: "ftp.guysports.co.uk"}))
	if err != nil {
		return err
	}
	defer func() {
		if err := ftpClient.Quit(); err != nil {
			fmt.Printf("error closing FTP connection %v\n", err)
		}
	}()

	if err := ftpClient.Login("guysports@guysports.co.uk", globals.FtpPassword); err != nil {
		return err
	}
	if err := ftpClient.ChangeDir("/public_html/guysports/players"); err != nil {
		return err
	}

	entries, err := ftpClient.List(".")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		fmt.Printf("%-40s %10d %s\n", entry.Name, entry.Size, entry.Time.Format("2006-01-02 15:04:05"))
	}
	return nil
}
