package log

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/deis/logger/storage"
	"github.com/vmihailenco/msgpack"
)

const (
	podPattern              = `(\w.*)-(\w.*)-(\w.*)-(\w.*)`
	controllerPattern       = `^(INFO|WARN|DEBUG|ERROR)\s+(\[(\S+)\])+:(.*)`
	controllerContainerName = "deis-controller"
	timeFormat              = "2006-01-02T15:04:05.000000000-07:00"
)

var (
	controllerRegex = regexp.MustCompile(controllerPattern)
	podRegex        = regexp.MustCompile(podPattern)
)

func handle(rawMessage []byte, storageAdapter storage.Adapter) error {
	message := new(Message)
	if err := json.Unmarshal(rawMessage, message); err != nil {
		message := new(MessageWithDockerString)
		if err := json.Unmarshal(rawMessage, message); err != nil {
			return err
		}
	}
	return processMessage(message, storageAdapter)
}

func handleMsgPack(rawMessage []byte, storageAdapter storage.Adapter) error {
	message := new(Message)
	if err := msgpack.Unmarshal(rawMessage, message); err != nil {
		message := new(MessageWithDockerString)
		if err := msgpack.Unmarshal(rawMessage, message); err != nil {
			return err
		}
	}
	return processMessage(message, storageAdapter)
}

func processMessage(message *Message, storageAdapter storage.Adapter) error {
	if fromController(message) {
		storageAdapter.Write(getApplicationFromControllerMessage(message), buildControllerLogMessage(message))
	} else {
		labels := message.Kubernetes.Labels
		storageAdapter.Write(labels["app"], buildApplicationLogMessage(message))
	}
	return nil
}

func HandleJsonTail(rawMessage []byte) (string, string) {
	message := new(Message)
	if err := json.Unmarshal(rawMessage, message); err != nil {
		message := new(MessageWithDockerString)
		if err := json.Unmarshal(rawMessage, message); err != nil {
			return "", ""
		}
	}
	return processTailMessage(message)
}

func HandleMsgPackTail(rawMessage []byte) (string, string) {
	message := new(Message)
	if err := msgpack.Unmarshal(rawMessage, message); err != nil {
		message := new(MessageWithDockerString)
		if err := msgpack.Unmarshal(rawMessage, message); err != nil {
			return "", ""
		}
	}
	return processTailMessage(message)
}

func processTailMessage(message *Message) (string, string) {

	if fromController(message) {
		return getApplicationFromControllerMessage(message), buildControllerLogMessage(message)
	} else {
		labels := message.Kubernetes.Labels
		return labels["app"], buildApplicationLogMessage(message)
	}
}

func fromController(message *Message) bool {
	matched, _ := regexp.MatchString(controllerContainerName, message.Kubernetes.ContainerName)
	if matched {
		matched, _ = regexp.MatchString(controllerPattern, message.Log)
	}
	return matched
}

func getApplicationFromControllerMessage(message *Message) string {
	return controllerRegex.FindStringSubmatch(message.Log)[3]
}

func buildControllerLogMessage(message *Message) string {
	l := controllerRegex.FindStringSubmatch(message.Log)
	return fmt.Sprintf("%s deis[controller]: %s %s",
		message.Time.Format(timeFormat),
		l[1],
		strings.Trim(l[4], " "))
}

func buildApplicationLogMessage(message *Message) string {
	p := podRegex.FindStringSubmatch(message.Kubernetes.PodName)
	tag := fmt.Sprintf(
		"%s.%s",
		message.Kubernetes.Labels["type"],
		message.Kubernetes.Labels["version"])
	if len(p) > 0 {
		tag = fmt.Sprintf("%s.%s", tag, p[len(p)-1])
	}
	return fmt.Sprintf("%s %s[%s]: %s",
		message.Time.Format(timeFormat),
		message.Kubernetes.Labels["app"],
		tag,
		message.Log)
}
