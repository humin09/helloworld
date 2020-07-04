package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main()  {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter text: ")
	s, _ := reader.ReadString('\n')
	s1:=strings.Split(s,"@tcp(")
	s2:=strings.Split(s1[0],":")
	user:=s2[0]
	passwd:=s2[1]
	s3:=strings.Split(s1[1],")/")
	host:=s3[0][:strings.Index(s3[0],":")]
	port:=s3[0][strings.Index(s3[0],":")+1:]
	db:=s3[1][:strings.Index(s3[1],"?")]
	a:=fmt.Sprintf("mysql -h%s -P%s -u%s -p%s -D%s --default-character-set=utf8",host,port,user,passwd,db)
	fmt.Println(a)
}
