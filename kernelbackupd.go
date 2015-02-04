package main

/* 
  Created by Beget Team
  License: GPLv2
*/

import "time"
import "log"
import "fmt"
import "bufio"
import "sync"
import "io"
import "os"
import "flag"
import "syscall"
import "strings"
import "path/filepath"
import "os/signal"

/*import "net/http"*/
import "marisa"

import "github.com/garyburd/redigo/redis"

//import "runtime/pprof"
/*
import "net/http"
import _ "net/http/pprof"
*/

const logPath = "/sys/kernel/debug/backup/log*"
const basePath = "/opt/disk1/kernel_backup_log/"

type KernelBackupd struct {
	mainChan      chan []byte
	die           bool
	dump          bool // force dump
	readers       sync.WaitGroup
	readInterval  time.Duration
	flushInterval time.Duration
	basePath      string

	redis_host string
	pool       *redis.Pool
	workers    sync.WaitGroup
}

func (self *KernelBackupd) readFrom(path string) {
	tick := time.Tick(self.readInterval)

	log.Printf("[reader][%s] start reading", path)
	for _ = range tick {
		file, err := os.Open(path)
		if err != nil {
			log.Print(err)
			continue
		}
		reader := bufio.NewReader(file)
		for {
			if self.die {
				log.Printf("[reader][%s] time to die, stopping reading", path)
				file.Close()
				goto out
			}
			line, err := reader.ReadBytes(byte('\n'))
			if err != nil {
				if err != io.EOF {
					log.Print(err)
				}
				break
			}
			select {
			case self.mainChan <- line:
			default:
				log.Printf("[reader][%s] parent buffer full, message lost", path)
			}
		}
		file.Close()
	}
out:
	self.readers.Done()
	log.Printf("[reader][%s] stopping", path)
}

func parse(line []byte) (login []byte, path []byte) {
	var ignored_paths = [...]string{"/home/mysql", "/home/tmpfs", "/home/system"}
	var matches = strings.SplitAfterN(string(line), " ", 3)
	if len(matches) < 3 {
		return
	}
	var path_components = strings.SplitAfterN(matches[2], "/", 5)

	if len(path_components) > 3 {
		path = []byte(strings.TrimRight(matches[2], "\n"))
		for _, prefix := range ignored_paths {
			if strings.HasPrefix(string(path), prefix) {
				return
			}
		}
		login = []byte(strings.TrimRight(path_components[3], "/"))
	}
	return
}

func (self *KernelBackupd) pusher(fromReader chan []byte) {
	timeout := time.Tick(self.flushInterval)
	defer self.workers.Done()

	for {
		if self.die {
			break
		}
		select {
		case line, ok := <-fromReader:
			if !ok {
				break
			}

			login, path := parse(line)
			if string(login) == "" {
				continue
			}
			conn := self.pool.Get()
			if conn != nil {
				conn.Do("SELECT", 1) // database 1
				key := fmt.Sprintf("kernelbackupd:%s:%s", time.Now().Format("20060102"), login)
				_, err := conn.Do("ZADD", key, 1, string(path))
				if err != nil {
					log.Print(err)
				}
				conn.Close()
			} else {
				log.Print("can't connect to redis")
			}
		case _ = <-timeout:
		}
	}
}

func (self *KernelBackupd) dailyDump(date time.Time, filename string) {
	var dateString = date.Format("20060102")
	var keyset = marisa.NewKeySet()

	self.workers.Add(1)
	defer self.workers.Done()

	log.Printf("[writer] dumping changes for %s", dateString)
	dirpath := filepath.Join(self.basePath, dateString)
	err := os.MkdirAll(dirpath, 0700)

	if err != nil {
		log.Printf("[writer][%s] can't create path %s: %v", dirpath, err)
		return
	}

	triepath := filepath.Join(dirpath, "log.trie")
	/*var file *os.File
	logpath := filepath.Join(dirpath, filename)

	file, err = os.OpenFile(logpath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)

	if err != nil {
		log.Printf("[writer][%s] can't create file %s: %v", logpath, err)
		return
	}
	writer := bufio.NewWriter(file)
	*/
	conn := self.pool.Get()
	conn.Do("SELECT", 1) // database 1
	keys, _ := redis.Strings(conn.Do("KEYS", fmt.Sprintf("kernelbackupd:%s:*", date.Format("20060102"))))
	log.Print("deleting keys for date: ", date.Format("20060102"))
	conn.Close()
	for _, key := range keys {
		conn = self.pool.Get()
		conn.Do("SELECT", 1) // database 1
		lines, _ := redis.Strings(conn.Do("ZREVRANGE", key, 0, -1))
		for _, line := range lines {
			/* fmt.Fprintln(writer, line) */
			keyset.Push(line, 1)
		}
		conn.Close()

		if date.Day() != time.Now().Day() {
		retry:
			conn = self.pool.Get()
			conn.Do("SELECT", 1) // database 1
			log.Printf("Deleting key %s", key)
			_, err := conn.Do("DEL", key)
			if err != nil {
				log.Print(err)
				goto retry
			}
			conn.Close()
		}
	}
	/*
		writer.Flush()
		file.Close()
	*/
	log.Printf("[writer] changes for %s dumped", dateString)

	log.Printf("[writer] building trie for %s", dateString)
	trie := marisa.NewTrie()
	trie.Build(*keyset, 0)
	trie.Save(triepath)
	log.Printf("[writer] trie for %s saved", dateString)
	trie.Free()
	keyset.Free()
	log.Printf("[writer] memory freed")
}

func (self *KernelBackupd) dumpTo(filename string) {
	const layout = "20060102"
	var day = time.Now()

	log.Printf("[writer][%s] starting", filename)
	for i := 0; i < 100; i++ {
		self.workers.Add(1)
		go self.pusher(self.mainChan)
	}
	for !self.die {
		time.Sleep(1 * time.Second)
		today := time.Now().Day()
		if day.Day() != today || self.dump {
			go self.dailyDump(day, filename)
			self.dump = false
			day = time.Now()
		}
	}
	log.Printf("[writer] waiting for workers")
	self.workers.Wait()
	log.Printf("[writer] stopped")
}

func (self *KernelBackupd) handle(signals chan os.Signal) {
	log.Print("[sighandler] starting")
	for s := range signals {
		switch s {
		case syscall.SIGINT:
			log.Println("[sighandler] SIGINT received: time to die set")
			self.die = true
		case syscall.SIGTERM:
			log.Println("[sighandler] SIGTERM received: time to die set")
			self.die = true
		case syscall.SIGUSR1:
			log.Println("[sighandler] SIGUSER1 received: forcing dump")
			self.dump = true
		default:
			log.Printf("[sighandler] signal %v received", s)
		}

	}
	log.Printf("[sighandler] stopping")
}

func isEnabled() bool {
	const sysctl_path = "/proc/sys/kernel/"

	files := []string{"fastvps_logging_root", "fastvps_logging_user"}

	for _, file := range files {
		f, err := os.Open(filepath.Join(sysctl_path, file))
		if err != nil {
			return false
		}
		reader := bufio.NewReader(f)
		line, err := reader.ReadString('\n')
		if err != nil || line[0] != '1' {
			return false
		}
	}
	return true
}

func (self *KernelBackupd) Run() {
	log.Printf("[main] checking if kernel log is enabled")
	if !isEnabled() {
		log.Printf("[main] kernel log disabled")
		return
	}
	log.Printf("[main] kernel log enabled")

	paths, err := filepath.Glob(logPath)

	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	go self.handle(signals)

	if err != nil {
		log.Print(err)
	}
	for _, path := range paths {
		self.readers.Add(1)
		go self.readFrom(path)
	}
	self.pool = &redis.Pool{
		MaxIdle:     200,
		MaxActive:   120,
		IdleTimeout: 30 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialTimeout("unix", self.redis_host, 5*time.Second, 5*time.Second, 5*time.Second)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	go self.reopenLog()
	self.mainChan = make(chan []byte, bufferSize*len(paths))
	/*
		http.HandleFunc("/changes/", func(w http.ResponseWriter, r *http.Request) {
			login := r.URL.Path[len("/changes/"):]
			log.Printf("List of changes for %s requested", login)
			conn := self.pool.Get()
			lines, _ := redis.Strings(conn.Do("ZREVRANGE", fmt.Sprintf("kernelbackupd:%s:%s", time.Now().Format("20060102"), login), 0, -1))
			for _, l := range lines {
				fmt.Fprintf(w, "%s\n", l)
			}
			conn.Close()
		})
		go http.ListenAndServe(":2988", nil)
	*/
	/*
		go func() {
			log.Println(http.ListenAndServe("localhost:1488", nil))
		}()
	*/
	self.dumpTo("log")
}

func (self *KernelBackupd) reopenLog() {
	tick := time.Tick(10 * time.Minute)
	var err error = nil
	var logfile *os.File
	const logpath string = "/var/log/kernelbackupd.log"

	for {
		if logfile != nil {
			logfile.Close()
		}
		logfile, err = os.OpenFile(logpath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)
		if err != nil {
			panic(fmt.Sprintf("cannot open %s", logpath))
		}
		log.SetOutput(logfile)
		<-tick
	}
}

func New() *KernelBackupd {
	self := new(KernelBackupd)
	//self.mainChan = make(chan []byte, 10000)
	self.readInterval = time.Duration(readInterval) * time.Second
	self.flushInterval = time.Duration(flushInterval) * time.Second
	self.basePath = basePath
	self.redis_host = "/var/run/redis/redis.sock"

	return self
}

var showVersion = false
var bufferSize = 10 * 1000 * 1000
var readInterval = 1
var flushInterval = 5

func init() {
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.IntVar(&bufferSize, "b", 10 * 1000 * 1000, "buffer size")
	flag.IntVar(&readInterval, "r", 5, "read kernel log every N seconds")
	flag.IntVar(&flushInterval, "f", 10, "flush kernel log every N seconds")
}

func main() {
	flag.Parse()
	if showVersion {
		fmt.Println("version: 0.6.1")
	} else {
		kernelbackupd := New()
		log.Print("starting")
		kernelbackupd.Run()
		log.Print("stopped")
	}
}
