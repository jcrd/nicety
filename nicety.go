// This project is licensed under the MIT License (see LICENSE).

package main

import (
    "bufio"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "os/exec"
    "os/signal"
    "os/user"
    "path"
    "path/filepath"
    "regexp"
    "runtime"
    "strconv"
    "strings"
    "sync/atomic"
    "syscall"
    "time"
)

var version = ""

type Rule struct {
    Name *string `json:"name"`
    CPUAffinity *string `json:"cpu_affinity"`
    Nice *int `json:"nice"`
    IOClass *string `json:"io_class"`
    IOPriority *int `json:"io_priority"`
    SchedPolicy *string `json:"sched_policy"`
    SchedPriority *int `json:"sched_priority"`
    Delay *int `json:"delay"`
}

type Rules map[string]Rule

var globalRules atomic.Value

var digitsRegexp = regexp.MustCompile("[[:digit:]]+")
var numCPU = runtime.NumCPU()
var logErr = log.New(os.Stderr, "", 0)

var verbose = false

func getRules() Rules {
    return globalRules.Load().(Rules)
}

func setRules(rs Rules) {
    globalRules.Store(rs)
}

func loadRule(path string) (r Rule, err error) {
    f, err := ioutil.ReadFile(path)

    if err != nil {
        return r, err
    }

    if err = json.Unmarshal(f, &r); err != nil {
        return r, err
    }

    if r.Name == nil {
        return r, fmt.Errorf("name: required key")
    }

    within := func(i int, min int, max int) bool {
        return !(i < min || i > max)
    }

    if r.CPUAffinity != nil {
        list := strings.Replace(*r.CPUAffinity, "-", ",", -1)
        for _, cpu := range strings.Split(list, ",") {
            i, err := strconv.Atoi(cpu)
            if err != nil || i >= numCPU {
                return r, fmt.Errorf("cpu_affinity: expected CPU 0-" +
                    strconv.Itoa(numCPU))
            }
        }
    }

    if r.Nice != nil {
        if !within(*r.Nice, -20, 19) {
            return r, fmt.Errorf("nice: expected number -20-19")
        }
    }

    if r.IOClass != nil {
        switch *r.IOClass {
        case "none", "realtime", "best-effort", "idle":
        default:
            return r, fmt.Errorf("io_class: expected one of " +
                "[none, realtime, best-effort, idle]")
        }
    }

    if r.IOPriority != nil {
        if !within(*r.IOPriority, 0, 7) {
            return r, fmt.Errorf("io_priority: expected number 0-7")
        }
    }

    if r.SchedPolicy != nil {
        switch *r.SchedPolicy {
        case "batch", "deadline", "fifo", "idle", "other", "rr":
        default:
            return r, fmt.Errorf("sched_policy: expected one of " +
                "[batch, deadline, fifo, idle, other, rr]")
        }
    }

    if r.SchedPriority != nil {
        if !within(*r.SchedPriority, 1, 99) {
            return r, fmt.Errorf("sched_priority: expected number 1-99")
        }
    }

    return r, nil
}

func loadRules(path string) Rules {
    files, err := ioutil.ReadDir(path)

    logRulesErr := log.New(os.Stderr, "[rules] ", 0)

    if err != nil {
        logRulesErr.Println(err)
        return nil
    }

    rules := make(map[string]Rule)

    for _, file := range files {
        n := file.Name()
        if filepath.Ext(n) != ".rules" {
            continue
        }
        p := filepath.Join(path, n)
        rule, err := loadRule(p)

        if err != nil {
            logRulesErr.Println(p + ": " + err.Error())
            continue
        }

        rules[*rule.Name] = rule

        if verbose {
            fmt.Println("[rules] loaded: " + p)
        }
    }

    if len(rules) > 0 {
        return rules
    }

    return nil
}

func (r Rule) apply(comm, pid string) {
    var pids []string

    tasks := func() []string {
        if pids == nil {
            files, err := ioutil.ReadDir(fmt.Sprintf("/proc/%s/task", pid))
            if err != nil {
                logErr.Println(err)
                pids[0] = pid
            } else {
                for _, file := range files {
                    pids = append(pids, file.Name())
                }
            }
        }
        return pids
    }

    printErr := func(name string, err error) {
        logErr.Printf("%s[%s] %s error: %v\n", comm, pid, name, err)
    }

    var msg []string
    appendMsg := func(format string, args ...interface{}) {
        if verbose {
            msg = append(msg, fmt.Sprintf(format, args...))
        }
    }

    if r.CPUAffinity != nil {
        err := exec.Command("taskset", "-a", "-c", *r.CPUAffinity,
            "-p", pid).Run()
        if err != nil {
            printErr("taskset", err)
        }
        appendMsg("taskset: %s", *r.CPUAffinity)
    }

    if r.Nice != nil {
        args := append([]string{strconv.Itoa(*r.Nice)}, tasks()...)
        err := exec.Command("renice", args...).Run()
        if err != nil {
            printErr("renice", err)
        }
        appendMsg("renice: %d", *r.Nice)
    }

    if r.IOClass != nil {
        args := []string{"-c", *r.IOClass}
        prio := "-"
        if r.IOPriority != nil {
            prio = strconv.Itoa(*r.IOPriority)
            args = append(args, "-n", prio)
        }
        args = append(args, "-p")
        args = append(args, tasks()...)
        err := exec.Command("ionice", args...).Run()
        if err != nil {
            printErr("ionice", err)
        }
        appendMsg("ionice: %s %s", *r.IOClass, prio)
    }

    if r.SchedPolicy != nil {
        prio := "0"
        if r.SchedPriority != nil {
            prio = strconv.Itoa(*r.SchedPriority)
        }
        pol := fmt.Sprintf("--%s", *r.SchedPolicy)
        err := exec.Command("chrt", "-a", pol, "--pid", prio, pid).Run()
        if err != nil {
            printErr("chrt", err)
        }
        appendMsg("chrt: %s %d", *r.SchedPolicy, prio)
    }

    if verbose && msg != nil {
        fmt.Printf("%s[%s] %s\n", comm, pid, strings.Join(msg, ", "))
    }
}

func getComm(pid string) (string, bool) {
    file := fmt.Sprintf("/proc/%s/comm", pid)

    if data, err := ioutil.ReadFile(file); err != nil {
        return "", false
    } else {
        return string(data[:len(data) - 1]), true
    }
}

func (rs Rules) match(s string) (r Rule, b bool) {
    for name, r := range rs {
        if name == s {
            return r, true
        }
    }
    return r, false
}

func (rs Rules) apply() {
    files, err := ioutil.ReadDir("/proc")
    if err != nil {
        logErr.Println(err)
        return
    }

    for _, file := range files {
        pid := file.Name()
        if !file.IsDir() || !digitsRegexp.MatchString(pid) {
            continue
        }
        if comm, ok := getComm(pid); ok {
            if rule, ok := rs.match(comm); ok {
                rule.apply(comm, pid)
            }
        }
    }
}

func parseText(text string) (string, string) {
    fields := strings.Fields(text)
    b := []byte(filepath.Base(fields[1]))
    if len(b) > 15 {
        b = b[:15]
    }
    return fields[0], string(b)
}

func main() {
    apply := flag.Bool("a", false, "Affect existing processes")
    rulesPath := flag.String("d", "/etc/nicety/rules.d",
        "Path to rules directory")
    delay := flag.Uint("s", 1, "Default delay in seconds")
    ver := flag.Bool("v", false, "Show version")
    flag.BoolVar(&verbose, "V", false, "Print additional output")

    flag.Parse()

    if *ver {
        fmt.Println(version)
        os.Exit(0)
    }

    if usr, err := user.Current(); err != nil {
        logErr.Fatal(err)
    } else if usr.Uid != "0" {
        logErr.Fatal("root permissions required")
    }

    rs := loadRules(*rulesPath)
    if rs == nil {
        logErr.Fatal("No valid rules exist; exiting")
    }

    if *apply {
        rs.apply()
    }

    setRules(rs)

    ctx, cancel := context.WithCancel(context.Background())
    extrace := exec.CommandContext(ctx, "/usr/bin/extrace", "-f", "-q")
    stdout, err := extrace.StdoutPipe()
    if err != nil {
        logErr.Fatal(err)
    }
    if err := extrace.Start(); err != nil {
        logErr.Fatal(err)
    }

    wait := make(chan struct{})

    go func() {
        extrace.Wait()
        close(wait)
    }()

    scanErr := make(chan error)
    scanText := make(chan string)

    go func() {
        scanner := bufio.NewScanner(stdout)
        for {
            for scanner.Scan() {
                scanText <- scanner.Text()
            }
            if err := scanner.Err(); err != nil {
                cancel()
                scanErr <- err
            }
        }
    }()

    reload := make(chan os.Signal, 1)
    signal.Notify(reload, syscall.SIGHUP)
    reapply := make(chan os.Signal, 1)
    signal.Notify(reapply, syscall.SIGUSR1)

    for {
        select {
        case text := <-scanText:
            pid, comm := parseText(text)
            if rule, ok := getRules().match(comm); ok {
                go func() {
                    s := *delay
                    if rule.Delay != nil {
                        s = uint(*rule.Delay)
                    }

                    time.Sleep(time.Duration(s) * time.Second)

                    if c, ok := getComm(pid); ok && c == path.Base(comm) {
                        rule.apply(comm, pid)
                    }
                }()
            }
        case err := <-scanErr:
            logErr.Fatal(err)
        case <-reload:
            rs := loadRules(*rulesPath)
            if rs != nil {
                setRules(rs)
            }
        case <-reapply:
            getRules().apply()
        case <-wait:
            os.Exit(128)
        }
    }
}
