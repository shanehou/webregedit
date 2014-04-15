package main

import (
	"bufio"
	"code.google.com/p/go.text/encoding/unicode"
	"code.google.com/p/go.text/transform"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	_ "log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type OutputNode struct {
	Id   bson.ObjectId                `json:"id"   bson:"_id"`
	Path string                       `json:"path" bson:"path"`
	Text string                       `json:"text" bson:"text"`
	Attr map[string]map[string]string `json:"attr" bson:"attr"`
	Chil bool                         `json:"children" bson:"children"`
}

type NodeEntry struct {
	Id   bson.ObjectId `json:"id"   bson:"_id"`
	Path string        `json:"path" bson:"path"`
	Text string        `json:"text" bson:"text"`
	Attr bson.M        `json:"attr" bson:"attr"`
	Chil bool          `json:"children" bson:"children"`
}

type NodeEntryList []NodeEntry

type importHandler struct {
	collection *mgo.Collection
}

type outputHandler struct {
	collection *mgo.Collection
}

type dropHandler struct {
	collection *mgo.Collection
}

type readHandler struct {
	collection *mgo.Collection
}

type nodeHandler struct {
	collection *mgo.Collection
}

type attrHandler struct {
	collection *mgo.Collection
}

func writeValueData(c *mgo.Collection, nodeId bson.ObjectId, value string, valueData string) (err error) {
	valueType := ""
	if strings.HasPrefix(valueData, `"`) && strings.HasSuffix(valueData, `"`) {
		valueType = "REG_SZ"
		valueData = strings.Trim(valueData, `"`)
	} else if strings.HasPrefix(valueData, "dword:") {
		valueType = "REG_DWORD"
		valueData = strings.TrimPrefix(valueData, "dword:")
	} else if strings.HasPrefix(valueData, "hex:") {
		valueType = "REG_BINARY"
		valueData = strings.TrimPrefix(valueData, "hex:")
	} else if strings.HasPrefix(valueData, "hex(") {
		switch string(valueData[4]) {
		case "0":
			valueType = "REG_NONE"
		case "1":
			valueType = "REG_SZ"
		case "2":
			valueType = "REG_EXPAND_SZ"
		case "7":
			valueType = "REG_MULTI_SZ"
		case "8":
			valueType = "REG_RESOURCE_LIST"
		case "9":
			valueType = "REG_FULL_RESOURCE_DESCRIPTOR"
		case "a":
			valueType = "REG_RESOURCE_REQUIREMENTS_LIST"
		case "b":
			valueType = "REG_QWORD"
		}
		valueData = valueData[7:]
	} else {
		err = errors.New("Unidentified type: " + valueData)
		return err
	}
	fmt.Println("type: " + valueType)
	fmt.Println(value + "=" + valueData)
	err = c.Update(bson.M{"_id": nodeId}, bson.M{"$set": bson.M{"attr." + value: bson.M{"data": valueData, "type": valueType}}})
	return err
}

func processRegistry(c *mgo.Collection, r io.Reader) {
	var err error
	var nodeId bson.ObjectId
	value := ""
	valueData := ""
	// var changeInfo *mgo.ChangeInfo
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		currentLine := scanner.Text()
		if currentLine == "" || strings.HasPrefix(currentLine, "Windows Registry Editor Version") {
			continue
		}
		if strings.HasPrefix(currentLine, "[") && strings.HasSuffix(currentLine, "]") {
			if value != "" && valueData != "" {
				err = writeValueData(c, nodeId, value, valueData)
				if err != nil {
					panic(err)
				}
				value = ""
				valueData = ""
			}
			nodeId = bson.NewObjectId()
			currentLine = strings.Trim(currentLine, "[]")
			lastPos := strings.LastIndex(currentLine, `\`)
			text := currentLine[lastPos+1 : len(currentLine)]
			path := `\` + currentLine + `\`
			query := c.Find(bson.M{"path": path, "text": text})
			exist, err := query.Count()
			if err != nil {
				panic(err)
			}
			if exist == 0 {
				_, err = c.Upsert(bson.M{"path": path}, bson.M{"_id": nodeId, "path": path, "text": text})
				if err != nil {
					panic(err)
				}
			} else {
				var existedNode NodeEntry
				err = query.One(&existedNode)
				if err != nil {
					panic(err)
				}
				nodeId = existedNode.Id
			}
			fmt.Println()
			fmt.Println("id: " + nodeId)
			fmt.Println("path: " + path)
		} else {
			re := regexp.MustCompile(`[^\\](\\\\)*"=`) // Some value names also contain `=`, or have `\"=` as end, or have `=` as start, so it's wrong to simply split current line with `=` or find `"="`.
			if strings.HasPrefix(currentLine, "@") {   // default value name
				if value != "" && valueData != "" {
					err = writeValueData(c, nodeId, value, valueData)
					if err != nil {
						panic(err)
					}
				}
				value = currentLine[:1]
				valueData = currentLine[2:]
			} else if re.MatchString(currentLine) {
				if value != "" && valueData != "" {
					err = writeValueData(c, nodeId, value, valueData)
					if err != nil {
						panic(err)
					}
				}
				split := re.FindStringIndex(currentLine)
				value = currentLine[:split[1]-1]
				valueData = strings.TrimSuffix(currentLine[split[1]:], `\`)
			} else {
				currentLine = strings.TrimSuffix(currentLine, `\`)
				currentLine = strings.TrimLeft(currentLine, ` `)
				valueData += currentLine
			}
		}
	}
}

func (h *importHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	invalid := errors.New("")
	fname := string(r.PostFormValue("registry"))
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		invalid = errors.New("File does not exists!")
	} else {
		fmt.Println("Importing file: " + fname)
		f, err := os.Open(fname)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		enc := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM)
		tf := enc.NewDecoder()
		reader := transform.NewReader(f, tf)
		processRegistry(h.collection, reader)
	}
	response, err := json.Marshal(bson.M{"err": invalid.Error()})
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (h *outputHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	invalid := errors.New("")
	var iterNode OutputNode
	iter := h.collection.Find(bson.M{}).Iter()
	f, err := os.OpenFile(r.PostFormValue("registry"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for iter.Next(&iterNode) {
		_, err = f.WriteString("[" + iterNode.Path[1:len(iterNode.Path)-1] + "]\n")
		if err != nil {
			panic(err)
		}
		for value, valueContent := range iterNode.Attr {
			switch valueContent["type"] {
			case "REG_SZ":
				_, err = f.WriteString(value + `="` + valueContent["data"] + `"`)
			case "REG_DWORD":
				_, err = f.WriteString(value + "=dword:" + valueContent["data"])
			case "REG_BINARY":
				_, err = f.WriteString(value + "=hex:" + valueContent["data"])
			case "REG_NONE":
				_, err = f.WriteString(value + "=hex(0):" + valueContent["data"])
			case "REG_EXPAND_SZ":
				_, err = f.WriteString(value + "=hex(2):" + valueContent["data"])
			case "REG_MULTI_SZ":
				_, err = f.WriteString(value + "=hex(7):" + valueContent["data"])
			case "REG_RESOURCE_LIST":
				_, err = f.WriteString(value + "=hex(8):" + valueContent["data"])
			case "REG_FULL_RESOURCE_DESCRIPTOR":
				_, err = f.WriteString(value + "=hex(9):" + valueContent["data"])
			case "REG_RESOURCE_REQUIREMENTS_LIST":
				_, err = f.WriteString(value + "=hex(a):" + valueContent["data"])
			case "REG_QWORD":
				_, err = f.WriteString(value + "=hex(b):" + valueContent["data"])
			default:
				_, err = f.WriteString(value + "=" + valueContent["data"])
			}
			if err != nil {
				panic(err)
			}
			_, err = f.WriteString("\n")
			if err != nil {
				panic(err)
			}
		}
	}
	err = iter.Err()
	if err != nil {
		panic(err)
	}
	response, err := json.Marshal(bson.M{"err": invalid.Error()})
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (h *dropHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.collection.DropCollection()
	if err != nil {
		panic(err)
	}
	response, err := json.Marshal(bson.M{"err": ""})
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (l NodeEntryList) detectChildren(c *mgo.Collection) (err error) {
	children := 0
	for i, _ := range l {
		children, err = c.Find(bson.M{"path": bson.M{"$regex": bson.RegEx{`^` + strings.Replace(l[i].Path, `\`, `\\`, -1) + `[^\\]*\\$`, ""}}}).Count()
		if children > 0 {
			l[i].Chil = true
		}
	}
	return err
}

func (h *readHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := h.collection
	var node NodeEntry
	var nodeList NodeEntryList
	var err error
	objectId := r.URL.Path[len("/read/"):]
	if objectId == "root" {
		err = c.Find(bson.M{"path": bson.M{"$regex": bson.RegEx{`^\\[^\\]*\\$`, ""}}}).All(&nodeList)
	} else {
		err = c.FindId(bson.ObjectIdHex(objectId)).One(&node)
		if err != nil {
			panic(err)
		}
		err = c.Find(bson.M{"path": bson.M{"$regex": bson.RegEx{`^` + strings.Replace(node.Path, `\`, `\\`, -1) + `[^\\]*\\$`, ""}}}).All(&nodeList)
	}
	if err != nil {
		panic(err)
	}
	err = nodeList.detectChildren(c)
	if err != nil {
		panic(err)
	}
	response, err := json.Marshal(nodeList)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (h *nodeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := h.collection
	var err error
	var node NodeEntry
	objectId := bson.ObjectIdHex(r.PostFormValue("objectId"))
	err = c.FindId(objectId).One(&node)
	if err != nil {
		panic(err)
	}
	text := r.PostFormValue("text")
	text = strings.Replace(text, `\`, "", -1)
	operation := r.URL.Path[len("/node/"):]
	invalid := errors.New("")
	fmt.Println()
	fmt.Println("Node operation: " + operation)
	fmt.Println("ObjectId: " + objectId)
	switch operation {
	case "create":
		newPath := node.Path + text + `\`
		objectId = bson.NewObjectId()
		newNode := NodeEntry{Id: objectId, Path: newPath, Text: text, Attr: bson.M{}, Chil: false}
		fmt.Println("New node: " + newNode.Path)
		err = c.Insert(newNode)
		if err != nil {
			panic(err)
		}
		err = c.UpdateId(node.Id, bson.M{"$set": bson.M{"children": true}})
		if err != nil {
			panic(err)
		}
	case "rename":
		textPos := strings.LastIndex(node.Path, node.Text)
		newPath := node.Path[:textPos] + text + `\`
		fmt.Println("Original node: " + node.Path)
		fmt.Println("New node: " + newPath)
		err = c.UpdateId(objectId, bson.M{"$set": bson.M{"text": text, "path": newPath}})
		if err != nil {
			panic(err)
		}
		var iterNode NodeEntry
		iter := c.Find(bson.M{"path": bson.M{"$regex": "^" + strings.Replace(node.Path, `\`, `\\`, -1) + ".+"}}).Iter()
		for iter.Next(&iterNode) {
			newPath = iterNode.Path[:len(node.Path)-len(node.Text)-1] + text + iterNode.Path[len(node.Path)-1:]
			err = c.UpdateId(iterNode.Id, bson.M{"$set": bson.M{"path": newPath}})
			if err != nil {
				panic(err)
			}
		}
		err = iter.Err()
		if err != nil {
			panic(err)
		}
	case "delete":
		fmt.Println("Remove node: " + node.Path)
		_, err = c.RemoveAll(bson.M{"path": bson.M{"$regex": "^" + strings.Replace(node.Path, `\`, `\\`, -1)}})
		if err != nil {
			panic(err)
		}
	default:
		invalid = errors.New("Invalid operation!")
	}
	response, err := json.Marshal(bson.M{"err": invalid.Error(), "id": objectId})
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (h *attrHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := h.collection
	var err error
	var node NodeEntry
	objectId := bson.ObjectIdHex(r.PostFormValue("objectId"))
	err = c.FindId(objectId).One(&node)
	if err != nil {
		panic(err)
	}
	operation := r.URL.Path[len("/attr/"):]
	fmt.Println()
	fmt.Println("Attribute operation: " + operation)
	fmt.Println("objectId: " + objectId)
	invalid := errors.New("")
	switch operation {
	case "add":
		value := r.PostFormValue("value")
		valueType := r.PostFormValue("valueType")
		valueData := r.PostFormValue("valueData")
		exist, err := c.Find(bson.M{"_id": objectId, "attr." + `"` + value + `"`: bson.M{"$exists": true}}).Count()
		if err != nil {
			panic(err)
		}
		if exist > 0 {
			invalid = errors.New("value already exists!")
		} else {
			fmt.Println("New value: " + value)
			fmt.Println("New value type: " + valueType)
			fmt.Println("New value data: " + valueData)
			err = c.UpdateId(objectId, bson.M{"$set": bson.M{"attr." + `"` + value + `"`: bson.M{"data": valueData, "type": valueType}}})
		}
		if err != nil {
			panic(err)
		}
	case "edit":
		value := r.PostFormValue("value")
		origValue := r.PostFormValue("orgiValue")
		valueData := r.PostFormValue("valueData")
		if origValue != value {
			exist, err := c.Find(bson.M{"_id": objectId, "attr." + `"` + value + `"`: bson.M{"$exists": true}}).Count()
			if err != nil {
				panic(err)
			}
			fmt.Println("Exists: " + string(exist))
			if exist > 0 {
				invalid = errors.New("value already exists!")
			} else {
				fmt.Println("Original value: " + origValue)
				fmt.Println("New value: " + value)
				fmt.Println("New value data: " + valueData)
				err = c.UpdateId(objectId, bson.M{"$rename": bson.M{"attr." + `"` + origValue + `"`: "attr." + `"` + value + `"`}})
				if err != nil {
					panic(err)
				}
			}
		}
		if origValue == "@" {
			err = c.UpdateId(objectId, bson.M{"$set": bson.M{"attr.@.data": valueData}})
		} else {
			err = c.UpdateId(objectId, bson.M{"$set": bson.M{"attr." + `"` + value + `".data`: valueData}})
		}
		if err != nil {
			panic(err)
		}
	case "delete":
		value := r.PostFormValue("value")
		fmt.Println("Delete value: " + value)
		err = c.UpdateId(objectId, bson.M{"$unset": bson.M{"attr." + `"` + value + `"`: ""}})
		if err != nil {
			panic(err)
		}
	default:
		invalid = errors.New("Invalid Operation!")
	}
	response, err := json.Marshal(bson.M{"err": invalid.Error()})
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func main() {
	fmt.Println("Web Windows Registry Editor Logging:")
	var err error
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	session.SetSafe(&mgo.Safe{})
	collection := session.DB("winreg").C("nodes")
	handlers := http.NewServeMux()
	handlers.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:]+"static/index.html")
	})
	handlers.HandleFunc("/doc", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/"+r.URL.Path[1:]+".html")
	})
	handlers.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	handlers.Handle("/import/", &importHandler{collection})
	handlers.Handle("/output/", &outputHandler{collection})
	handlers.Handle("/drop/", &dropHandler{collection})
	handlers.Handle("/read/", &readHandler{collection})
	handlers.Handle("/node/", &nodeHandler{collection})
	handlers.Handle("/attr/", &attrHandler{collection})
	server := &http.Server{":2048", handlers, 60 * 1000 * 1000 * 1000, 60 * 1000 * 1000 * 1000, 0, nil, nil}
	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
