package main

import (
	"encoding/json" // 提供用于编码和解码 JSON 数据的功能。
	"log"
	"net/http"
)

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		log.Println("Responding with 5XX error:", msg)
	}
	type errResponse struct {
		Error string `json:"error"`
	}

	respondWithJSON(w, code, errResponse{Error: msg})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	/*
		json.Marshal(payload) 是 Go 语言标准库中的 encoding/json 包提供的一个函数，
		用于将 Go 语言中的数据结构（如结构体、map、切片等）序列化（或编码）为 JSON 格式的字节切片
	*/
	dat, err := json.Marshal(payload)
	// dat 是 字节切片类型: []byte
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v", payload)
		w.WriteHeader(500)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}
