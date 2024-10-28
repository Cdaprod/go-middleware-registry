// File: internal/ui/messages.go
package ui

// Message types for various UI components
type (
    // Docker-related messages
    dockerMsg struct {
        Type        string // "error", "success", "info", "status"
        Message     string
        ContainerID string
        Data        interface{}
    }

    // Container-related messages
    containerMsg struct {
        ID     string
        Output string
        Status string
        Error  error
    }

    // UI state messages
    clearMessageMsg struct{}
    
    loadingMsg struct {
        Active  bool
        Message string
    }

    // Operation messages
    operationCompleteMsg struct {
        Success bool
        Message string
        Data    interface{}
    }

    execFinishedMsg struct {
        Error error
        Data  interface{}
    }

    // Menu-related messages
    menuMsg struct {
        Type    string
        Action  string
        ItemID  string
        Data    interface{}
    }

    // Status update messages
    statusMsg struct {
        Type    string
        Message string
    }
)

// Message type constants
const (
    // Docker message types
    MsgTypeError    = "error"
    MsgTypeSuccess  = "success"
    MsgTypeInfo     = "info"
    MsgTypeWarning  = "warning"
    MsgTypeStatus   = "status"
    
    // Container states
    ContainerStarted = "started"
    ContainerStopped = "stopped"
    ContainerError   = "error"
    
    // Menu actions
    MenuActionSelect = "select"
    MenuActionClose  = "close"
    MenuActionUpdate = "update"
)