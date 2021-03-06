/*
    Simple Netstat implementation.
    Get data from /proc/net/tcp and /proc/net/udp and
    and parse /proc/[0-9]/fd/[0-9].

    Author: Rafael Santos <rafael@sourcecode.net.br>
*/

package GOnetstat

import (
    "fmt"
    "io/ioutil"
    "strings"
    "os"
    "os/user"
    "strconv"
    "path/filepath"
    "regexp"
)


const (
    PROC_TCP = "/proc/net/tcp"
    PROC_UDP = "/proc/net/udp"
    PROC_TCP6 = "/proc/net/tcp6"
    PROC_UDP6 = "/proc/net/udp6"

    SOCKET_STATE_ESTABLISHED = "01"
    SOCKET_STATE_SYN_SENT    = "02"
    SOCKET_STATE_SYN_RECV    = "03"
    SOCKET_STATE_FIN_WAIT1   = "04"
    SOCKET_STATE_FIN_WAIT2   = "05"
    SOCKET_STATE_TIME_WAIT   = "06"
    SOCKET_STATE_CLOSE       = "07"
    SOCKET_STATE_CLOSE_WAIT  = "08"
    SOCKET_STATE_LAST_ACK    = "09"
    SOCKET_STATE_LISTEN      = "0A"
    SOCKET_STATE_CLOSING     = "0B"
)

var (
  STATE = map[string]string {
      SOCKET_STATE_ESTABLISHED: "ESTABLISHED",
      SOCKET_STATE_SYN_SENT:    "SYN_SENT",
      SOCKET_STATE_SYN_RECV:    "SYN_RECV",
      SOCKET_STATE_FIN_WAIT1:   "FIN_WAIT1",
      SOCKET_STATE_FIN_WAIT2:   "FIN_WAIT2",
      SOCKET_STATE_TIME_WAIT:   "TIME_WAIT",
      SOCKET_STATE_CLOSE:       "CLOSE",
      SOCKET_STATE_CLOSE_WAIT:  "CLOSE_WAIT",
      SOCKET_STATE_LAST_ACK:    "LAST_ACK",
      SOCKET_STATE_LISTEN:      "LISTEN",
      SOCKET_STATE_CLOSING:     "CLOSING",
  }
)


type Process struct {
    User         string
    Name         string
    Pid          string
    Exe          string
    State        string
    IP           string
    Port         int64
    ForeignIP    string
    ForeignPort  int64
}


func getData(t string) []string {
    // Get data from tcp or udp file.

    var procT string

    if t == "tcp" {
        procT = PROC_TCP
    } else if t == "udp" {
        procT = PROC_UDP
    } else if t == "tcp6" {
        procT = PROC_TCP6
    } else if t == "udp6" {
        procT = PROC_UDP6
    } else {
        fmt.Printf("%s is a invalid type, tcp and udp only!\n", t)
        os.Exit(1)
    }


    data, err := ioutil.ReadFile(procT)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    lines := strings.Split(string(data), "\n")

    // Return lines without Header line and blank line on the end
    return lines[1:len(lines) - 1]

}


func hexToDec(h string) int64 {
    // convert hexadecimal to decimal.
    d, err := strconv.ParseInt(h, 16, 32)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    return d
}


func convertIP(ip string) string {
    // Convert the ipv4 to decimal. Have to rearrange the ip because the
    // default value is in little Endian order.

    var out string

    // Check ip size if greater than 8 is a ipv6 type
    if len(ip) > 8 {
        i := []string{ ip[30:32],
                        ip[28:30],
                        ip[26:28],
                        ip[24:26],
                        ip[22:24],
                        ip[20:22],
                        ip[18:20],
                        ip[16:18],
                        ip[14:16],
                        ip[12:14],
                        ip[10:12],
                        ip[8:10],
                        ip[6:8],
                        ip[4:6],
                        ip[2:4],
                        ip[0:2]}
        out = fmt.Sprintf("%v%v:%v%v:%v%v:%v%v:%v%v:%v%v:%v%v:%v%v",
                            i[14], i[15], i[13], i[12],
                            i[10], i[11], i[8], i[9],
                            i[6],  i[7], i[4], i[5],
                            i[2], i[3], i[0], i[1])

    } else {
        i := []int64{ hexToDec(ip[6:8]),
                       hexToDec(ip[4:6]),
                       hexToDec(ip[2:4]),
                       hexToDec(ip[0:2]) }

       out = fmt.Sprintf("%v.%v.%v.%v", i[0], i[1], i[2], i[3])
    }
   return out
}


func findPid(inode string) string {
    // Loop through all fd dirs of process on /proc to compare the inode and
    // get the pid.

    pid := "-"

    d, err := filepath.Glob("/proc/[0-9]*/fd/[0-9]*")
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    re := regexp.MustCompile(inode)
    for _, item := range(d) {
        path, err := os.Readlink(item)
        if err != nil {
          continue
        }
        out := re.FindString(path)
        if len(out) != 0 {
            pid = strings.Split(item, "/")[2]
        }
    }
    return pid
}


func getProcessExe(pid string) (string, error) {
    exe := fmt.Sprintf("/proc/%s/exe", pid)
    return os.Readlink(exe)
}


func getProcessName(exe string) string {
    n := strings.Split(exe, "/")
    name := n[len(n) -1]
    return strings.Title(name)
}


func getUser(uid string) string {
    u, err := user.LookupId(uid)
    if err != nil {
      return "-"
    }
    return u.Username
}


func removeEmpty(array []string) []string {
    // remove empty data from line
    var newArray []string
    for _, i := range(array) {
        if i != "" {
           newArray = append(newArray, i)
        }
    }
    return newArray
}


func netstat(t string) []Process {
    // Return a array of Process with Name, Ip, Port, State .. etc
    // Require Root acess to get information about some processes.

    var Processes []Process

    data := getData(t)

    for _, line := range(data) {

        // local ip and port
        lineArray := removeEmpty(strings.Split(strings.TrimSpace(line), " "))
        ipPort := strings.Split(lineArray[1], ":")
        ip := convertIP(ipPort[0])
        port := hexToDec(ipPort[1])

        // foreign ip and port
        fipPort := strings.Split(lineArray[2], ":")
        fip := convertIP(fipPort[0])
        fport := hexToDec(fipPort[1])

        state := lineArray[3]
        // uid := getUser(lineArray[7])
        // pid := findPid(lineArray[9])
        // exe, err := getProcessExe(pid)
        // name := "-"
        // if err != nil {
        //   fmt.Printf("Couldn't find process exec located at /proc/%s/exe\n", pid)
        // } else {
        //   name = getProcessName(exe)
        // }

        p := Process{
          // User: uid,
          // Name: name,
          // Pid: pid,
          // Exe: exe,
          State: state,
          IP: ip,
          Port: port,
          ForeignIP: fip,
          ForeignPort: fport,
        }

        Processes = append(Processes, p)

    }

    return Processes
}


func Tcp() []Process {
    // Get a slice of Process type with TCP data
    data := netstat("tcp")
    return data
}


func Udp() []Process {
    // Get a slice of Process type with UDP data
    data := netstat("udp")
    return data
}


func Tcp6() []Process {
    // Get a slice of Process type with TCP6 data
    data := netstat("tcp6")
    return data
}


func Udp6() []Process {
    // Get a slice of Process type with UDP6 data
    data := netstat("udp6")
    return data
}
