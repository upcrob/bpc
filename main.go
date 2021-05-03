package main

import (
	"fmt"
	"github.com/mitchellh/go-ps"
	"strconv"
	"os"
	"os/user"
	"io/ioutil"
	"runtime"
	"os/exec"
	"strings"
)

type TaskRecord struct {
	id int
	pid int
	command string
}

func main() {
	createEnvironment()

	args := os.Args[1:]

	if len(args) < 1 {
		fmt.Println("bpc <command>")
		fmt.Println("  clean - remove process history")
		fmt.Println("  history - show the execution history")
		fmt.Println("  show <id> - show the output from the process with the given id")
		fmt.Println("  stop <id> - stop the background process with the given id")
		fmt.Println("  start <command/id> - start the given command as a background process")
		fmt.Println("  status - show the current running processes")
	} else if args[0] == "status" {
		printRecordList(getActiveRecords())
	} else if args[0] == "history" {
		printRecordList(readHistory())
	} else if args[0] == "start" {
		command := strings.Join(args[1:], " ")
		if v, err := strconv.Atoi(command); err == nil {
			history := readHistory()
			for i := 0; i < len(history); i++ {
				record := history[i]
				if record.id == v {
					command = record.command
					break
				}
			}
		}
		id := nextHistoryId()
		pid := run(command, id)
		writeTaskRecord(id, pid, command)
		writeActiveRecord(id, pid, command)
		fmt.Println(strconv.Itoa(id) + "\t" + strconv.Itoa(pid) + "\t" + command)
	} else if args[0] == "stop" {
		if len(args) < 2 {
			fmt.Println("job id expected")
			os.Exit(1)
		}
		id, _ := strconv.Atoi(args[1])
		killProc(id)
	} else if args[0] == "show" {
		if len(args) < 2 {
			fmt.Println("job id expected")
			os.Exit(1)
		}
		id, _ := strconv.Atoi(args[1])
		printOutput(id)
	} else if args[0] == "clean" {
		removeHistory()
	} else {
		fmt.Println("invalid command")
	}
}

func getHomePath() string {
	user, _ := user.Current()
	return user.HomeDir + "/.bpc"
}

func getHistoryPath() string {
	return getHomePath() + "/history"
}

func getActivePath() string {
	return getHomePath() + "/active"
}

func removeHistory() {
	home := getHomePath()

	os.Remove(getHistoryPath())

	files, err := ioutil.ReadDir(home)
	if err != nil {
		fmt.Println("failed to read history")
		os.Exit(1)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".out") {
			os.Remove(home + "/" + file.Name())
		}
	}
}

func killProc(id int) {
	active := getActiveRecords()
	for _, rec := range active {
		if rec.id == id {
			command := "kill"
			pid := strconv.Itoa(rec.pid)
			shell := pid
			if runtime.GOOS == "windows" {
				command = "cmd"
				shell = "/c taskkill /f /pid " + pid
			}

			cmd := exec.Command(command, shell)
			_, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Println(err)
				fmt.Println("failed to stop process")
			}
			return
		}
	}
}

func printOutput(id int) {
	content, err := ioutil.ReadFile(getHomePath() + "/" + strconv.Itoa(id) + ".out")
	if err != nil {
		fmt.Println("failed to read output")
		os.Exit(1)
	}

	fmt.Println(string(content))
}

func printRecordList(list []TaskRecord) {
	fmt.Println("id\tpid\tcommand")
	for _, rec := range list {
		fmt.Println(strconv.Itoa(rec.id) + "\t" + strconv.Itoa(rec.pid) + "\t" + rec.command)
	}
}

func getActiveRecords() []TaskRecord {
	processes, _ := ps.Processes()
	procMap := make(map[int]bool)
	for _, proc := range processes {
		procMap[proc.Pid()] = true
	}

	content, err := ioutil.ReadFile(getActivePath())
	if err != nil {
		fmt.Println("Failed to read history file")
		os.Exit(1)
	}

	lines := strings.Split(string(content), "\n")
	records := make([]TaskRecord, 0)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		record := TaskRecord{}
		parts := strings.Split(line, "\t")
		record.id, _ = strconv.Atoi(parts[0])
		record.pid, _ = strconv.Atoi(parts[1])
		record.command = parts[2]

		if (procMap[record.pid]) {
			records = append(records, record)
		}
	}

	// rewrite active file
	var data string = ""
	for _, record := range records {
		data = data + strconv.Itoa(record.id) + "\t" + strconv.Itoa(record.pid) + "\t" + record.command + "\n"
	}
	ioutil.WriteFile(getActivePath(), []byte(data), 0644)

	return records
}

func writeTaskRecord(id int, pid int, command string) int {
	f, err := os.OpenFile(getHistoryPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open history file")
		os.Exit(1)
	}
	defer f.Close()

	f.WriteString(strconv.Itoa(id) + "\t" + strconv.Itoa(pid) + "\t" + command + "\n")
	return id
}

func nextHistoryId() int {
	records := readHistory()
	id := 1
	if len(records) > 0 {
		id = records[len(records) - 1].id + 1
	}
	return id
}

func writeActiveRecord(id int, pid int, command string) {
	f, err := os.OpenFile(getActivePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open history file")
		os.Exit(1)
	}
	defer f.Close()

	f.WriteString(strconv.Itoa(id) + "\t" + strconv.Itoa(pid) + "\t" + command + "\n")
}

func readHistory() []TaskRecord {
	content, err := ioutil.ReadFile(getHistoryPath())
	if err != nil {
		fmt.Println("Failed to read history file")
		os.Exit(1)
	}

	lines := strings.Split(string(content), "\n")
	records := make([]TaskRecord, 0)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		record := TaskRecord{}
		parts := strings.Split(line, "\t")
		record.id, _ = strconv.Atoi(parts[0])
		record.pid, _ = strconv.Atoi(parts[1])
		record.command = parts[2]
		records = append(records, record)
	}
	return records
}

func run(command string, recordId int) int {
	shell := "bash"
	arg := "-c"
	if runtime.GOOS == "windows" {
		shell = "cmd"
		arg = "/c"
	}
	
	cmd := exec.Command(shell, arg, command)
	handle, ferr := os.OpenFile(getHomePath() + "/" + strconv.Itoa(recordId) + ".out", os.O_RDWR|os.O_CREATE, 0644)
	if ferr != nil {
		fmt.Println(ferr)
	}
	cmd.Stdout = handle
	cmd.Stderr = handle

	err := cmd.Start()
	if err != nil {
		fmt.Println(err)
	}
	return cmd.Process.Pid
}

func createEnvironment() {
	home := getHomePath()
	running := getActivePath()
	history := getHistoryPath()

	if !fileExists(home) {
		os.Mkdir(home, 0755)
	}

	if !fileExists(running) {
		ioutil.WriteFile(getActivePath(), []byte(""), 0644)
	}

	if !fileExists(history) {
		ioutil.WriteFile(getHistoryPath(), []byte(""), 0644)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}