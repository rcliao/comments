# Component Responsibility Matrix

## Core Components

### 1. Comment Parser
**Primary Responsibility**: Transform raw markdown with CriticMarkup into structured data

**Responsibilities**:
- Tokenize markdown content into lexical units
- Identify and extract CriticMarkup comment syntax
- Build Abstract Syntax Tree (AST) from tokens
- Extract comment metadata (author, id, line, timestamp)
- Handle malformed syntax gracefully with error recovery
- Support incremental parsing for performance

**Interfaces**:
- Input: Raw markdown string
- Output: DocumentAST with embedded Comment objects
- Errors: ParseError array with positions

**Dependencies**:
- None (pure transformation)

**Verification**:
- Parses valid CriticMarkup correctly
- Recovers from malformed syntax
- Preserves document structure
- Maintains position accuracy

---

### 2. Document Manager
**Primary Responsibility**: Manage document lifecycle and persistence

**Responsibilities**:
- Load documents from filesystem
- Save documents with atomic writes
- Track document changes via checksums
- Manage backup creation and rotation
- Handle file encoding detection/conversion
- Coordinate with Parser for content processing

**Interfaces**:
- Input: File paths, Document objects
- Output: Document content, save confirmations
- Events: DocumentChanged, SaveCompleted

**Dependencies**:
- Comment Parser (for initial load)
- Position Tracker (for save serialization)

**Verification**:
- Files persist correctly
- Atomic writes prevent corruption
- Backups created on major changes
- Change detection accurate

---

### 3. Position Tracker
**Primary Responsibility**: Maintain comment position integrity during document edits

**Responsibilities**:
- Map between offset/line/column representations
- Calculate position deltas from edit operations
- Update comment positions after edits
- Detect and resolve position conflicts
- Maintain position indices for fast lookup
- Handle overlapping edit scenarios

**Interfaces**:
- Input: EditOperation, current positions
- Output: Updated position map, conflicts
- Events: PositionsUpdated

**Dependencies**:
- Document Manager (for content access)

**Verification**:
- Positions remain accurate after edits
- Conflicts detected and reported
- Performance scales with document size
- Indices stay synchronized

---

### 4. Terminal UI Controller
**Primary Responsibility**: Manage terminal-based user interface

**Responsibilities**:
- Handle keyboard input events
- Manage split-pane layout
- Control view focus and navigation
- Coordinate scroll synchronization
- Buffer and optimize terminal output
- Manage UI state transitions

**Interfaces**:
- Input: Key events, terminal dimensions
- Output: Rendered terminal frames
- Events: UIStateChanged, UserAction

**Dependencies**:
- Document Renderer (for document view)
- Comment Panel (for comment view)
- Command Processor (for commands)

**Verification**:
- Responsive to user input
- Smooth scrolling and navigation
- Correct layout calculations
- Efficient rendering

---

### 5. Document Renderer
**Primary Responsibility**: Convert document AST to terminal-displayable format

**Responsibilities**:
- Render markdown with syntax highlighting
- Apply comment indicators inline
- Handle line wrapping and truncation
- Generate line numbers and gutters
- Apply search/selection highlights
- Optimize for terminal capabilities

**Interfaces**:
- Input: DocumentAST, viewport, highlights
- Output: Rendered line array with styles
- Configuration: Theme, display options

**Dependencies**:
- None (pure rendering)

**Verification**:
- Markdown renders correctly
- Comments visible inline
- Highlights applied accurately
- Performance acceptable

---

### 6. Comment Panel
**Primary Responsibility**: Display and manage comment threads

**Responsibilities**:
- Render threaded comment hierarchy
- Handle comment filtering and search
- Display comment metadata and reactions
- Manage comment input/edit modes
- Show thread context and navigation
- Update in real-time with changes

**Interfaces**:
- Input: Comment array, filters, viewport
- Output: Rendered comment panel
- Events: CommentSelected, ReplyInitiated

**Dependencies**:
- Thread Manager (for hierarchy)

**Verification**:
- Threads display correctly
- Filtering works as expected
- Real-time updates visible
- Navigation intuitive

---

### 7. Thread Manager
**Primary Responsibility**: Organize comments into conversational threads

**Responsibilities**:
- Build thread hierarchy from comments
- Track thread participants and state
- Sort comments by timestamp/relevance
- Calculate thread metrics (depth, count)
- Handle thread resolution/archival
- Maintain read/unread state per user

**Interfaces**:
- Input: Flat comment array
- Output: Hierarchical thread structure
- Events: ThreadUpdated, ThreadResolved

**Dependencies**:
- None (pure organization)

**Verification**:
- Hierarchy built correctly
- Sorting maintains order
- Metrics accurate
- State transitions valid

---

### 8. LLM Orchestrator
**Primary Responsibility**: Coordinate LLM provider interactions

**Responsibilities**:
- Select appropriate provider based on availability
- Build context from document and comments
- Manage request queuing and rate limiting
- Handle provider-specific adaptations
- Coordinate response streaming
- Implement fallback strategies

**Interfaces**:
- Input: Query, context parameters, provider config
- Output: Response stream or completed text
- Events: StreamStarted, ChunkReceived, StreamComplete

**Dependencies**:
- Context Builder (for context prep)
- Provider Adapters (for API calls)
- Stream Handler (for responses)

**Verification**:
- Provider selection works
- Context within limits
- Streaming functions correctly
- Fallbacks activate on failure

---

### 9. Context Builder
**Primary Responsibility**: Prepare optimal context for LLM requests

**Responsibilities**:
- Extract relevant document sections
- Include pertinent comments and threads
- Calculate token counts and limits
- Apply context window strategies
- Format context for provider requirements
- Cache frequently-used contexts

**Interfaces**:
- Input: Document, selection, parameters
- Output: LLMContext object
- Metrics: Token count, relevance score

**Dependencies**:
- Document Manager (for content)
- Thread Manager (for comment context)

**Verification**:
- Context relevant to query
- Token limits respected
- Caching improves performance
- Format correct for providers

---

### 10. Provider Adapters
**Primary Responsibility**: Abstract provider-specific LLM APIs

**Responsibilities**:
- Translate requests to provider format
- Handle authentication and credentials
- Manage connection and retries
- Parse provider responses
- Report provider-specific errors
- Track usage metrics

**Interfaces**:
- Input: Normalized request
- Output: Provider response
- Configuration: API keys, endpoints

**Dependencies**:
- None (network only)

**Verification**:
- Each provider works correctly
- Error handling comprehensive
- Metrics tracked accurately
- Authentication secure

---

### 11. Stream Handler
**Primary Responsibility**: Manage streaming LLM responses

**Responsibilities**:
- Buffer incoming chunks
- Reassemble partial responses
- Detect stream completion/errors
- Provide backpressure control
- Handle connection interruptions
- Coordinate UI preview updates

**Interfaces**:
- Input: Response stream
- Output: Chunks, completion signal
- Events: ChunkReady, StreamError

**Dependencies**:
- UI Controller (for preview)

**Verification**:
- Chunks assembled correctly
- Errors detected promptly
- Backpressure prevents overflow
- UI updates smooth

---

### 12. Command Processor
**Primary Responsibility**: Parse and execute user commands

**Responsibilities**:
- Parse command syntax and arguments
- Validate command parameters
- Route to appropriate handlers
- Maintain command history
- Provide command completion hints
- Execute command sequences

**Interfaces**:
- Input: Command string
- Output: Execution result
- Events: CommandExecuted

**Dependencies**:
- All components (for execution)

**Verification**:
- Commands parse correctly
- Validation prevents errors
- History maintained
- Completion helpful

---

### 13. Search Engine
**Primary Responsibility**: Find content within documents and comments

**Responsibilities**:
- Execute text and regex searches
- Search across documents and comments
- Rank results by relevance
- Highlight search matches
- Provide search navigation
- Cache search indices

**Interfaces**:
- Input: Search query, scope
- Output: Matched positions, snippets
- Configuration: Search options

**Dependencies**:
- Document Manager (for content)
- Comment Parser (for structure)

**Verification**:
- Searches find all matches
- Regex patterns work
- Performance acceptable
- Highlights accurate

---

### 14. Session Manager
**Primary Responsibility**: Maintain application session state

**Responsibilities**:
- Track open documents
- Preserve user preferences
- Maintain command history
- Store UI state per document
- Handle session persistence
- Manage multi-document navigation

**Interfaces**:
- Input: Session events
- Output: Session state
- Storage: Session file

**Dependencies**:
- Document Manager (for documents)
- UI Controller (for state)

**Verification**:
- State persists correctly
- Preferences applied
- History maintained
- Navigation preserved

---

### 15. Notification System
**Primary Responsibility**: Alert users to relevant events

**Responsibilities**:
- Queue notifications
- Filter by relevance/preferences
- Display in appropriate UI location
- Track notification state
- Handle notification actions
- Respect notification settings

**Interfaces**:
- Input: Event notifications
- Output: UI notifications
- Configuration: User preferences

**Dependencies**:
- UI Controller (for display)

**Verification**:
- Notifications appear timely
- Filtering respects preferences
- Actions execute correctly
- State tracked accurately

---

## Component Interaction Patterns

### Synchronous Interactions
- Parser → Document Manager (parsing on load)
- UI Controller → Renderers (frame generation)
- Command Processor → Target Components (execution)

### Asynchronous Interactions
- Document Manager → File System (I/O operations)
- LLM Orchestrator → Providers (API calls)
- Stream Handler → UI Controller (chunk updates)

### Event-Driven Interactions
- Position Tracker → Components (position updates)
- Document Manager → Components (change notifications)
- Thread Manager → Comment Panel (thread updates)

### Data Flow Patterns
1. **Read Path**: File → Parser → Document Manager → UI
2. **Write Path**: UI → Document Manager → Serializer → File
3. **LLM Path**: Context Builder → Orchestrator → Provider → Stream Handler → UI
4. **Search Path**: Query → Search Engine → Results → UI Highlights

## Verification Matrix

| Component | Unit Tests | Integration Tests | Performance Tests | User Tests |
|-----------|------------|------------------|-------------------|------------|
| Comment Parser | Syntax validation | Document parsing | Large file parsing | Error recovery |
| Document Manager | I/O operations | Save/load cycle | Concurrent access | Data integrity |
| Position Tracker | Position math | Edit tracking | Bulk updates | Accuracy |
| Terminal UI | Event handling | Layout rendering | Render performance | Responsiveness |
| Document Renderer | Markdown rendering | Syntax highlighting | Long documents | Visual correctness |
| Comment Panel | Thread display | Real-time updates | Many comments | Usability |
| Thread Manager | Hierarchy building | Thread operations | Deep threads | Organization |
| LLM Orchestrator | Provider selection | Full request cycle | Rate limiting | Response quality |
| Context Builder | Context extraction | Token counting | Large contexts | Relevance |
| Provider Adapters | API translation | Network handling | Timeout handling | Reliability |
| Stream Handler | Chunk assembly | Streaming flow | Backpressure | Smoothness |
| Command Processor | Command parsing | Command execution | Command chains | Discoverability |
| Search Engine | Search algorithms | Result ranking | Index performance | Result quality |
| Session Manager | State persistence | Multi-document | State size | Restoration |
| Notification System | Event filtering | Display logic | Queue handling | User experience |