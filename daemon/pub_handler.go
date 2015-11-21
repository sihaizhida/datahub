package daemon

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/asiainfoLDP/datahub/cmd"
	"github.com/asiainfoLDP/datahub/ds"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type Sys struct {
	Supplystyle string `json:"supply_style"`
}
type Label struct {
	Ssys Sys `json:"sys"`
}
type ic struct {
	AccessType string `json:"itemaccesstype"`
	Comment    string `json:"comment"`
	Slabel     Label  `json:"label"`
}

func pubItemHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Println(r.URL.Path, "(pub dataitem)")

	reqBody, _ := ioutil.ReadAll(r.Body)
	pub := ds.PubPara{}
	if err := json.Unmarshal(reqBody, &pub); err != nil {
		WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorUnmarshal, "pub dataitem error while unmarshal reqBody")
		return
	}
	if CheckDataPoolExist(pub.Datapool) == false {
		WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorUnmarshal,
			fmt.Sprintf("datapool %s not exist, please check.", pub.Datapool))
		return
	}

	icpub := ic{AccessType: pub.Accesstype, Comment: pub.Comment}
	isys := Sys{Supplystyle: "batch"}
	icpub.Slabel = Label{Ssys: isys}

	body, err := json.Marshal(icpub)
	if err != nil {
		s := "pub dataitem error while marshal icpub struct"
		log.Println(s)
		WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorMarshal, s)
		return
	}
	log.Println(string(body))

	log.Println("daemon: connecting to", DefaultServer+r.URL.Path)
	req, err := http.NewRequest("POST", DefaultServer+r.URL.Path, bytes.NewBuffer(body))
	if len(loginAuthStr) > 0 {
		req.Header.Set("Authorization", loginAuthStr)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s := "pub dataitem service unavailable"
		WriteHttpResultWithoutData(w, http.StatusServiceUnavailable, cmd.ErrorServiceUnavailable, s)
		return
	}
	defer resp.Body.Close()

	//Get server result
	rbody, _ := ioutil.ReadAll(resp.Body)
	log.Println(resp.StatusCode, string(rbody))

	repo := ps.ByName("repo")
	item := ps.ByName("item")

	if resp.StatusCode == 200 {
		err := MkdirForDataItem(repo, item, pub.Datapool)
		if err != nil {
			RollBackItem(repo, item)
			WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorInsertItem,
				fmt.Sprintf("Mkdir error! %s", err.Error()))
		} else {
			err = InsertItemToDb(repo, item, pub.Datapool)
			if err != nil {
				RollBackItem(repo, item)
				WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorInsertItem,
					"Insert dataitem to datapool error, please check it immediately!")
			} else {
				WriteHttpResultWithoutData(w, http.StatusOK, cmd.ResultOK, "OK")
			}
		}
	} else {

		result := ds.Result{}
		err = json.Unmarshal(rbody, &result)
		if err != nil {
			s := "pub dataitem error while unmarshal server response"
			log.Println(s)
			WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorUnmarshal, s)
			return
		}
		log.Println(resp.StatusCode, result.Msg)
		WriteHttpResultWithoutData(w, resp.StatusCode, result.Code, result.Msg)
	}

	return

}

func pubTagHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Println(r.URL.Path, "(pub tag)")

	reqBody, _ := ioutil.ReadAll(r.Body)
	pub := ds.PubPara{}
	if err := json.Unmarshal(reqBody, &pub); err != nil {
		WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorUnmarshal, "pub tag error while unmarshal reqBody")
		return
	}
	if len(pub.Detail) == 0 {
		WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorUnmarshal, "tag detail is not found")
		return
	}

	repo := ps.ByName("repo")
	item := ps.ByName("item")
	tag := ps.ByName("tag")

	var NeedCopy bool
	//get destfilepath and check whether repo/dataitem has been published
	DestFilePath, err := CheckTagExistAndGetDestFilePath(repo, item, tag)
	if err != nil {
		WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorUnmarshal, err.Error())
		return
	}
	splits := strings.Split(pub.Detail, "/")
	FileName := splits[len(splits)-1]
	DestFullPathFileName := DestFilePath + "/" + FileName
	if len(splits) == 1 {
		if isFileExists(DestFullPathFileName) == false {
			WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorFileNotExist, DestFullPathFileName+" not found")
			return
		}
		NeedCopy = false
	} else {
		if isFileExists(pub.Detail) == false {
			WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorFileNotExist, pub.Detail+" not found")
			return
		}
		NeedCopy = true
	}

	body, err := json.Marshal(&struct {
		Commnet string `json:"comment"`
	}{
		pub.Comment})
	if err != nil {
		s := "pub tag error while marshal struct"
		log.Println(s)
		WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorMarshal, s)
		return
	}

	log.Println("daemon: connecting to", DefaultServer+r.URL.Path)
	req, err := http.NewRequest("POST", DefaultServer+r.URL.Path, bytes.NewBuffer(body))
	if len(loginAuthStr) > 0 {
		req.Header.Set("Authorization", loginAuthStr)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s := "pub tag service unavailable"
		WriteHttpResultWithoutData(w, http.StatusServiceUnavailable, cmd.ErrorServiceUnavailable, s)
		return
	}
	defer resp.Body.Close()

	//Get server result
	rbody, _ := ioutil.ReadAll(resp.Body)
	log.Println(resp.StatusCode, string(rbody))

	if resp.StatusCode == 200 {
		if NeedCopy {
			count, err := CopyFile(pub.Detail, DestFullPathFileName)
			if err != nil {
				RollBackTag(repo, item, tag)
				WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorInsertItem,
					"Copy file to datapool error, permission denied or path!")
				return
			}
			log.Printf("Copy %d bytes from %s to %s", count, pub.Detail, DestFullPathFileName)
		}
		err = InsertPubTagToDb(repo, item, tag, FileName)
		if err != nil {
			RollBackTag(repo, item, tag)
			WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorInsertItem,
				"Insert dataitem to datapool error, please check it immediately!")
		} else {
			WriteHttpResultWithoutData(w, http.StatusOK, cmd.ResultOK, "OK")
		}
	} else {

		result := ds.Result{}
		err = json.Unmarshal(rbody, &result)
		if err != nil {
			s := "pub dataitem error while unmarshal server response"
			log.Println(s)
			WriteHttpResultWithoutData(w, http.StatusBadRequest, cmd.ErrorUnmarshal, s)
			return
		}
		log.Println(resp.StatusCode, result.Msg)
		WriteHttpResultWithoutData(w, resp.StatusCode, result.Code, result.Msg)
	}

	return

}

func WriteHttpResultWithoutData(w http.ResponseWriter, httpcode, errorcode int, msg string) {
	w.WriteHeader(httpcode)
	respbody, _ := json.Marshal(&struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}{
		errorcode,
		msg})
	w.Write(respbody)
}

func MkdirForDataItem(repo, item, datapool string) (err error) {
	dpconn := GetDataPoolDpconn(datapool)
	if len(dpconn) != 0 {
		err = os.MkdirAll(dpconn+datapool+repo+item, 0755)
		return err
	} else {
		return errors.New(fmt.Sprintf("dpconn is not found for datapool %s", datapool))
	}
	return nil
}

func RollBackItem(repo, item string) {
	//Delete /repository/repo/item
}

func CheckTagExistAndGetDestFilePath(repo, item, tag string) (filepath string, err error) {
	exist, err := CheckTagExist(repo, item, tag)
	if err != nil {
		return "", err
	}
	if exist == true {
		return "", errors.New("Tag already exist.")
	}
	dpname, dpconn := GetDpNameAndDpConn(repo, item, tag)
	if len(dpname) == 0 || len(dpconn) == 0 {
		return "", errors.New("dpname or dpconn not found.")
	}
	return
}

func RollBackTag(repo, item, tag string) {
	//Delete /repository/repo/item
}

func CopyFile(src, des string) (w int64, err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		log.Println(err)
	}
	defer srcFile.Close()

	desFile, err := os.Create(des)
	if err != nil {
		log.Println(err)
	}
	defer desFile.Close()

	return io.Copy(desFile, srcFile)
}