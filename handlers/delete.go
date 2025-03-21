package handlers

import (
    "LunaTransfer/config"
    "LunaTransfer/common"
    "LunaTransfer/utils"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "github.com/gorilla/mux"
)

func DeleteFile(w http.ResponseWriter, r *http.Request) {
    username, ok := common.GetUsernameFromContext(r.Context())
    if !ok {
        http.Error(w, "Not logged in", http.StatusUnauthorized)
        return
    }

    vars := mux.Vars(r)
    filePath, ok := vars["filename"]
    if !ok || filePath == "" {
        http.Error(w, "File path required", http.StatusBadRequest)
        return
    }
    filePath = strings.Replace(filePath, "%2F", "/", -1)
    cleanPath := filepath.Clean(filePath)
    if strings.Contains(cleanPath, "..") {
        utils.LogError("DELETE_ERROR", fmt.Errorf("path traversal attempt"), username, "Attempted path traversal")
        http.Error(w, "Invalid file path", http.StatusBadRequest)
        return
    }

    appConfig, err := config.LoadConfig()
    if err != nil {
        utils.LogError("DELETE_ERROR", err, username, "Failed to load configuration")
        http.Error(w, "Server configuration error", http.StatusInternalServerError)
        return
    }
    userDir := filepath.Join(appConfig.StorageDirectory, username)
    fullPath := filepath.Join(userDir, cleanPath)
    fileInfo, err := os.Stat(fullPath)
    if os.IsNotExist(err) {
        http.Error(w, "File or directory not found", http.StatusNotFound)
        return
    }
    var isDir bool
    if fileInfo.IsDir() {
        isDir = true
        err = os.RemoveAll(fullPath)
    } else {
        isDir = false
        err = os.Remove(fullPath)
    }

    if err != nil {
        utils.LogError("DELETE_ERROR", err, username, fmt.Sprintf("Failed to delete %s", cleanPath))
        http.Error(w, "Failed to delete file or directory", http.StatusInternalServerError)
        return
    }

    if isDir {
        utils.LogSystem("DIRECTORY_DELETED", username, r.RemoteAddr, fmt.Sprintf("Deleted directory: %s", cleanPath))
    } else {
        utils.LogFileTransfer("DELETE", cleanPath, username, r.RemoteAddr, 0)
        go utils.NotifyFileDeleted(username, cleanPath)
    }

    w.Header().Set("Content-Type", "application/json")
    var message string
    if isDir {
        message = "Directory deleted successfully"
    } else {
        message = "File deleted successfully"
    }
    response := map[string]interface{}{
        "success": true,
        "message": message,
        "path":    cleanPath,
        "isDir":   isDir,
    }
    json.NewEncoder(w).Encode(response)
}

// DeleteFileJSON handles file/directory deletion using JSON request body
func DeleteFileJSON(w http.ResponseWriter, r *http.Request) {
    // Get the user from context
    username, ok := common.GetUsernameFromContext(r.Context())
    if !ok {
        http.Error(w, "Not logged in", http.StatusUnauthorized)
        return
    }

    // Parse the JSON body
    var request struct {
        Path string `json:"path"`
    }
    
    err := json.NewDecoder(r.Body).Decode(&request)
    if err != nil {
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }
    
    if request.Path == "" {
        http.Error(w, "Path is required", http.StatusBadRequest)
        return
    }
    
    // Clean the path and prevent directory traversal
    filePath := request.Path
    cleanPath := filepath.Clean(filePath)
    if strings.Contains(cleanPath, "..") {
        utils.LogError("DELETE_ERROR", fmt.Errorf("path traversal attempt"), username, "Attempted path traversal")
        http.Error(w, "Invalid path", http.StatusBadRequest)
        return
    }
    
    // Load configuration - this is what was missing
    appConfig, err := config.LoadConfig()
    if err != nil {
        utils.LogError("DELETE_ERROR", err, username, "Failed to load configuration")
        http.Error(w, "Server configuration error", http.StatusInternalServerError)
        return
    }
    
    userDir := filepath.Join(appConfig.StorageDirectory, username)
    fullPath := filepath.Join(userDir, cleanPath)
    
    // Check if the file exists
    fileInfo, err := os.Stat(fullPath)
    if err != nil {
        if os.IsNotExist(err) {
            http.Error(w, "File or directory not found", http.StatusNotFound)
        } else {
            utils.LogError("DELETE_ERROR", err, username, fmt.Sprintf("Error checking %s", cleanPath))
            http.Error(w, "Error checking file", http.StatusInternalServerError)
        }
        return
    }
    
    // Determine if it's a directory
    var isDir bool
    if fileInfo.IsDir() {
        isDir = true
        err = os.RemoveAll(fullPath)
    } else {
        isDir = false
        err = os.Remove(fullPath)
    }
    
    if err != nil {
        utils.LogError("DELETE_ERROR", err, username, fmt.Sprintf("Failed to delete %s", cleanPath))
        http.Error(w, "Failed to delete file or directory", http.StatusInternalServerError)
        return
    }
    
    // Log the operation
    if isDir {
        utils.LogSystem("DIRECTORY_DELETED", username, r.RemoteAddr, fmt.Sprintf("Deleted directory: %s", cleanPath))
    } else {
        utils.LogFileTransfer("DELETE", cleanPath, username, r.RemoteAddr, 0)
        go utils.NotifyFileDeleted(username, cleanPath)
    }
    
    w.Header().Set("Content-Type", "application/json")
    var message string
    if isDir {
        message = "Directory deleted successfully"
    } else {
        message = "File deleted successfully"
    }
    
    response := map[string]interface{}{
        "success": true,
        "message": message,
        "path":    cleanPath,
        "isDir":   isDir,
    }
    json.NewEncoder(w).Encode(response)
}
