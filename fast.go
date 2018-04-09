package main

import (
	"io"
	"bufio"
	"encoding/json"
	"sync"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sort"
)

// глобальные переменные запрещены
// cgo запрещен

//easyjson:json
type User struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"company"`
	Country  string   `json:"country"`
	Email    string   `json:"email"`
	Hits     []net.IP `json:"hits"`
	Job      string   `json:"job"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
	pos      int      `json:"-"`
}

type WriteStruct struct {
	Index int
	Data  []byte
}

func Fast(in io.Reader, out io.Writer, networks []string) {
	// сюда писать код
	//inChan := make(chan []byte, 0)
	toNetworkChan := make(chan User, 5)
	toBrowserChan := make(chan User, 5)
	writerChan := make(chan WriteStruct, 5)
	var wg, secWg, thirdWg sync.WaitGroup
	//wg.Add(1)
	//go processEntry(inChan, toNetworkChan, &wg)
	secWg.Add(10)
	for i := 0; i < 10; i++ {
		go processNetwork(toNetworkChan, toBrowserChan, networks, &secWg)
	}
	thirdWg.Add(10)
	for i := 0; i < 10; i++ {
		go processBrowser(toBrowserChan, writerChan, &thirdWg)
	}
	wg.Add(1)
	go processWrite(writerChan, out, &wg)

	sc := bufio.NewScanner(in)
	i := 0

	for sc.Scan() {
		i++
		var user User
		user.pos = i
		json.Unmarshal(sc.Bytes(), &user)
		toNetworkChan <- user

	}
	close(toNetworkChan)

	secWg.Wait()
	close(toBrowserChan)

	thirdWg.Wait()
	close(writerChan)

	wg.Wait()
}

//func processEntry(inChan chan []byte, outChan chan User, wg *sync.WaitGroup) {
//	i := 0
//	for data := range inChan {
//		a := data
//		user := User{pos: i}
//		json.Unmarshal(a, &user)
//		//i++
//		outChan <- user
//	}
//	close(outChan)
//	wg.Done()
//}

func processNetwork(inChan chan User, outChan chan User, networks []string, wg *sync.WaitGroup) {

	nwSlice := make([]*net.IPNet, len(networks))
	i := 0
	for _, n := range networks {
		_, nw, _ := net.ParseCIDR(n)
		nwSlice[i] = nw
		i++
	}
	//var ip net.IP
	var c int
NwLoop:
	for u := range inChan {
		c = 0
		for _, hit := range u.Hits {
			hit.DefaultMask()
			//ip = net.ParseIP(hit)
			for _, nw := range nwSlice {
				if nw.Contains(hit) {
					c++
				}
			}
			if c == 3 {
				outChan <- u
				continue NwLoop
			}
		}
	}
	//close(outChan)
	wg.Done()
}

func processBrowser(inChan chan User, outChan chan WriteStruct, wg *sync.WaitGroup) {

	browserRegex := regexp.MustCompile(`Chrome/(60.0.3112.90|52.0.2743.116|57.0.2987.133)`)

	var c int
BrLoop:
	for u := range inChan {
		c = 0
		for _, b := range u.Browsers {
			if browserRegex.MatchString(b) {
				c++
			}
			if c == 3 {
				outChan <- WriteStruct{Data: []byte("[" + strconv.Itoa(u.pos) + "] " + u.Name + " <" + strings.Replace(u.Email, `@`, ` [at] `, 1) + ">\n"), Index: u.pos}
				continue BrLoop
			}
		}
	}
	//close(outChan)
	wg.Done()
}

func processWrite(inChan chan WriteStruct, writer io.Writer, wg *sync.WaitGroup) {
	writeMap := make(map[int][]byte)

	indexes := make([]int, 0)
	for uData := range inChan {
		writeMap[uData.Index] = uData.Data
		indexes = append(indexes, uData.Index)
	}
	sort.Ints(indexes)

	writer.Write([]byte("Total: " + strconv.Itoa(len(writeMap)) + "\n"))
	//writer.Write(data)
	for _, i := range indexes {
		writer.Write(writeMap[i])
	}
	wg.Done()
}
