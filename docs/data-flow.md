# Data Flow Specification

## System Overview

The commenting system processes markdown documents with inline comments, provides interactive terminal UI, and integrates with LLM providers for AI-assisted writing.

## Core Data Flows

### 1. Document Loading and Display

```mermaid
graph TD
    A[User Opens Document] --> B[File System Read]
    B --> C[Raw Markdown Content]
    C --> D[Comment Parser]
    D --> E{Parse Comments}
    E -->|Success| F[Document AST]
    E -->|Error| G[Parse Error Handler]
    F --> H[Comment Extractor]
    H --> I[Comments Collection]
    F --> J[Document Renderer]
    I --> K[Comment Threader]
    K --> L[Threaded Comments]
    J --> M[Document View]
    L --> N[Comment Panel View]
    M --> O[Terminal UI Compositor]
    N --> O
    O --> P[Split Pane Display]
    G --> Q[Error Display]
```

### 2. Comment Creation Flow

```mermaid
graph TD
    A[User Initiates Comment] --> B{Input Method}
    B -->|Keyboard Shortcut| C[Comment Input Mode]
    B -->|Command| C
    C --> D[Capture Comment Text]
    D --> E[Get Current Position]
    E --> F[Generate Comment ID]
    F --> G[Create Comment Object]
    G --> H[Validate Comment]
    H --> I{Valid?}
    I -->|Yes| J[Calculate Insert Position]
    I -->|No| K[Validation Error]
    J --> L[Comment Serializer]
    L --> M[CriticMarkup Format]
    M --> N[Document Updater]
    N --> O[Position Tracker Update]
    O --> P[File Writer]
    P --> Q[Persist to Disk]
    Q --> R[Update UI Views]
    K --> S[Display Error]
```

### 3. LLM Integration Flow

```mermaid
graph TD
    A[User LLM Request] --> B[Command Parser]
    B --> C[Extract Query]
    C --> D[Context Builder]
    D --> E[Gather Document Context]
    D --> F[Gather Comment Context]
    D --> G[Gather Selection Context]
    E --> H[Context Aggregator]
    F --> H
    G --> H
    H --> I[LLM Request Builder]
    I --> J{Provider Selection}
    J -->|Claude| K[Claude Adapter]
    J -->|OpenAI| L[OpenAI Adapter]
    J -->|Local| M[Local Model Adapter]
    K --> N[API Request]
    L --> N
    M --> N
    N --> O[Stream Handler]
    O --> P[Response Chunker]
    P --> Q[UI Stream Display]
    Q --> R{User Decision}
    R -->|Approve| S[Response Processor]
    R -->|Reject| T[Discard Response]
    R -->|Edit| U[Edit Mode]
    S --> V[Comment Creator]
    V --> W[Insert into Document]
    U --> V
```

### 4. Comment Position Tracking During Edits

```mermaid
graph TD
    A[Document Edit Event] --> B[Edit Detector]
    B --> C[Calculate Edit Delta]
    C --> D{Edit Type}
    D -->|Insert| E[Shift Positions Forward]
    D -->|Delete| F[Shift Positions Backward]
    D -->|Replace| G[Complex Position Update]
    E --> H[Position Updater]
    F --> H
    G --> H
    H --> I[Iterate Comments]
    I --> J{Comment Position}
    J -->|Before Edit| K[No Change]
    J -->|After Edit| L[Update Position]
    J -->|Within Edit| M[Handle Overlap]
    K --> N[Keep Original]
    L --> O[Apply Delta]
    M --> P{Resolution Strategy}
    P -->|Move| Q[Relocate Comment]
    P -->|Delete| R[Mark for Deletion]
    P -->|Split| S[Split Comment]
    Q --> T[Update Comment Positions]
    R --> T
    S --> T
    T --> U[Reserialize Document]
    U --> V[Save Changes]
```

### 5. Thread Management Flow

```mermaid
graph TD
    A[New Reply] --> B[Parent Comment Lookup]
    B --> C{Parent Exists?}
    C -->|Yes| D[Create Reply Object]
    C -->|No| E[Error: Invalid Parent]
    D --> F[Add to Thread]
    F --> G[Update Thread Metadata]
    G --> H[Sort by Timestamp]
    H --> I[Update Thread View]
    I --> J[Notify Participants]
    J --> K[Persist Thread State]
```

## Error Handling Paths

### Parse Error Recovery

```mermaid
graph TD
    A[Parse Error] --> B{Error Type}
    B -->|Malformed Comment| C[Extract Raw Text]
    B -->|Missing Metadata| D[Use Defaults]
    B -->|Position Error| E[Best Effort Position]
    C --> F[Create Warning]
    D --> F
    E --> F
    F --> G[Continue Parsing]
    G --> H[Display with Warnings]
```

### LLM Failure Handling

```mermaid
graph TD
    A[LLM Request Fails] --> B{Failure Type}
    B -->|Network| C[Retry with Backoff]
    B -->|Rate Limit| D[Queue Request]
    B -->|Auth| E[Prompt Credentials]
    B -->|Timeout| F[Offer Cancellation]
    C --> G{Retry Count}
    G -->|< Max| H[Retry Request]
    G -->|>= Max| I[Fallback Provider]
    I --> J{Fallback Available?}
    J -->|Yes| K[Switch Provider]
    J -->|No| L[Display Error]
```

## State Synchronization

```mermaid
stateDiagram-v2
    [*] --> Idle
    Idle --> Loading: Open Document
    Loading --> Displaying: Parse Success
    Loading --> Error: Parse Failure
    Displaying --> Editing: User Edit
    Displaying --> Commenting: Add Comment
    Commenting --> Saving: Submit Comment
    Saving --> Displaying: Save Success
    Saving --> Error: Save Failure
    Editing --> Saving: Save Changes
    Error --> Idle: Reset
    Displaying --> LLMProcessing: LLM Request
    LLMProcessing --> Streaming: Response Started
    Streaming --> Approving: Stream Complete
    Approving --> Saving: Approve Response
    Approving --> Displaying: Reject Response
```

## Component Responsibilities

### Parser Component
- Tokenize markdown content
- Identify comment markers
- Build document AST
- Extract comment metadata
- Handle malformed syntax gracefully

### Document Manager
- File I/O operations
- Change detection
- Position tracking
- Serialization/deserialization
- Backup management

### UI Controller
- Keyboard event handling
- View composition
- Focus management
- Layout calculations
- Scroll synchronization

### LLM Orchestrator
- Provider selection
- Context preparation
- Request queuing
- Response streaming
- Error recovery

### Position Tracker
- Calculate text offsets
- Map line/column to offset
- Track edit deltas
- Update comment anchors
- Handle position conflicts

## Performance Considerations

### Async Operations
- File I/O: Always async
- LLM requests: Always async with streaming
- Parsing: Async for large files (>1MB)
- UI rendering: Sync with buffering

### Caching Strategy
- Cache parsed AST
- Cache comment threads
- Cache LLM context windows
- Invalidate on document change

### Batch Operations
- Batch position updates
- Batch UI redraws
- Batch file writes (with debouncing)