# Verification Criteria

## Component Verification Specifications

### 1. Comment Parser

#### Functional Verification
```yaml
test_cases:
  valid_syntax:
    input: "{>>[@alice:id123:5:2024-01-01T12:00:00Z] This is a comment <<}"
    expected:
      - comment_extracted: true
      - author: "alice"
      - id: "id123"
      - line: 5
      - text: "This is a comment"

  malformed_recovery:
    input: "{>>[@alice:id123 Missing fields"
    expected:
      - partial_parse: true
      - error_reported: true
      - graceful_degradation: true

  nested_comments:
    input: "Text {>>[@bob:1:1:2024] Outer {>>[@alice:2:1:2024] Inner <<} comment <<}"
    expected:
      - comment_count: 2
      - proper_nesting: true

  escaped_content:
    input: "{>>[@user:id:1:2024] Code: `{>>` should not parse <<}"
    expected:
      - literal_preservation: true
      - no_nested_parse: true
```

#### Performance Verification
- Parse 1MB document in < 500ms
- Incremental parse updates in < 50ms
- Memory usage < 10x document size
- Support documents with 10,000+ comments

#### Error Handling Verification
- Recover from 10 different malformed syntax patterns
- Preserve document integrity on parse failure
- Report error positions accurately
- Continue parsing after errors

---

### 2. Document Manager

#### Functional Verification
```yaml
test_cases:
  atomic_saves:
    scenario: "Concurrent save operations"
    verification:
      - no_data_corruption: true
      - last_write_wins: true
      - backup_created: true

  encoding_detection:
    inputs: [UTF-8, UTF-16, ASCII, Latin-1]
    expected:
      - correct_detection: true
      - proper_conversion: true
      - no_data_loss: true

  change_detection:
    operations:
      - modify_content
      - add_comment
      - delete_comment
    expected:
      - checksum_updated: true
      - dirty_flag_set: true
      - change_event_fired: true
```

#### Data Integrity Verification
- File content matches memory after save
- Backups restorable without data loss
- Checksums detect all changes
- File locks prevent corruption

#### Resilience Verification
- Recover from disk full errors
- Handle permission denied gracefully
- Survive process crashes (via temp files)
- Manage network drive disconnections

---

### 3. Position Tracker

#### Functional Verification
```yaml
test_cases:
  simple_insertion:
    edit: {type: insert, position: 50, text: "new text"}
    comments_before: [{id: "c1", position: 45}, {id: "c2", position: 55}]
    expected:
      - c1_position: 45  # unchanged
      - c2_position: 63  # shifted by 8

  deletion_overlap:
    edit: {type: delete, start: 40, end: 60}
    comments_before: [{id: "c1", position: 50}]
    expected:
      - c1_status: "conflicted"
      - resolution_options: [move_to_40, delete]

  complex_replacement:
    edit: {type: replace, start: 30, end: 50, new_text: "x"}
    comments_before: [{id: "c1", position: 25}, {id: "c2", position: 45}, {id: "c3", position: 55}]
    expected:
      - c1_position: 25  # unchanged
      - c2_status: "conflicted"
      - c3_position: 36  # shifted by -19
```

#### Accuracy Verification
- Position calculations match byte offsets
- Line/column mappings remain correct
- Multi-byte character handling accurate
- Line ending variations handled (CRLF/LF)

#### Performance Verification
- Update 1000 positions in < 100ms
- Batch updates more efficient than individual
- Index lookups O(log n)
- Memory overhead < 1KB per comment

---

### 4. Terminal UI Controller

#### Functional Verification
```yaml
test_cases:
  keyboard_navigation:
    inputs: [arrow_keys, page_up_down, home_end]
    expected:
      - cursor_moves_correctly: true
      - viewport_scrolls_appropriately: true
      - focus_changes_properly: true

  split_view_management:
    operations:
      - resize_panes
      - switch_focus
      - synchronize_scroll
    expected:
      - layout_maintains_proportions: true
      - content_remains_visible: true
      - no_rendering_artifacts: true

  mode_transitions:
    transitions:
      - normal_to_insert
      - insert_to_command
      - command_to_search
    expected:
      - state_preserved: true
      - indicators_updated: true
      - keybindings_switched: true
```

#### Responsiveness Verification
- Keystroke to action < 50ms
- Smooth scrolling at 60fps
- No flicker during updates
- Terminal resize handled < 100ms

#### Compatibility Verification
- Works in 80x24 minimum terminal
- Supports 256 colors and true color
- Degrades gracefully to 16 colors
- Functions without mouse support

---

### 5. LLM Orchestrator

#### Functional Verification
```yaml
test_cases:
  provider_fallback:
    scenario: "Primary provider fails"
    expected:
      - fallback_activated: true
      - context_preserved: true
      - user_notified: true

  context_size_management:
    input: {document_size: 10000_tokens, limit: 4000}
    expected:
      - context_truncated_intelligently: true
      - most_relevant_kept: true
      - token_count_accurate: true

  streaming_response:
    verification:
      - chunks_arrive_in_order: true
      - no_data_loss: true
      - backpressure_handled: true
```

#### Integration Verification
- All configured providers reachable
- Authentication tokens valid
- Rate limits respected
- Timeout handling works

#### Quality Verification
- Response relevant to context
- Formatting preserved in responses
- Code blocks handled correctly
- Markdown syntax maintained

---

### 6. Thread Manager

#### Functional Verification
```yaml
test_cases:
  thread_construction:
    comments: [
      {id: "1", parent: null},
      {id: "2", parent: "1"},
      {id: "3", parent: "2"},
      {id: "4", parent: "1"}
    ]
    expected:
      thread_structure:
        - id: "1"
          replies: ["2", "4"]
          depth: 0
        - id: "2"
          replies: ["3"]
          depth: 1
        - id: "3"
          replies: []
          depth: 2
        - id: "4"
          replies: []
          depth: 1

  orphan_handling:
    comments: [{id: "1", parent: "non_existent"}]
    expected:
      - orphan_detected: true
      - fallback_to_root: true
```

#### Consistency Verification
- Thread hierarchy always valid
- No circular dependencies
- Orphaned comments handled
- Depth limits enforced

---

### 7. Search Engine

#### Functional Verification
```yaml
test_cases:
  text_search:
    query: "TODO"
    expected:
      - all_occurrences_found: true
      - case_sensitivity_optional: true
      - word_boundaries_respected: true

  regex_search:
    query: "\\b[A-Z]{2,}\\b"  # Find acronyms
    expected:
      - regex_compiled: true
      - matches_accurate: true
      - groups_captured: true

  comment_search:
    filters: {author: "alice", date_from: "2024-01-01"}
    expected:
      - filters_applied: true
      - results_sorted: true
      - pagination_works: true
```

#### Performance Verification
- First results in < 100ms
- Search 10MB document in < 1s
- Incremental results display
- Index updates efficiently

---

## End-to-End Verification Scenarios

### Scenario 1: Document Collaboration Flow
```yaml
steps:
  1. open_document:
      verify:
        - document_loaded
        - comments_parsed
        - ui_rendered

  2. add_comment:
      verify:
        - comment_appears_inline
        - position_tracked
        - file_updated

  3. reply_to_comment:
      verify:
        - thread_created
        - hierarchy_displayed
        - participants_tracked

  4. edit_document:
      verify:
        - positions_updated
        - comments_remain_attached
        - no_data_loss

  5. save_and_reload:
      verify:
        - persistence_complete
        - state_restored
        - comments_intact
```

### Scenario 2: LLM Assistance Flow
```yaml
steps:
  1. select_text:
      verify:
        - selection_highlighted
        - context_captured

  2. request_llm_help:
      verify:
        - context_built
        - provider_selected
        - request_sent

  3. stream_response:
      verify:
        - chunks_displayed
        - preview_updated
        - no_ui_blocking

  4. approve_suggestion:
      verify:
        - content_inserted
        - comment_created
        - position_correct

  5. continue_editing:
      verify:
        - llm_comment_preserved
        - normal_flow_resumed
```

### Scenario 3: Large Document Handling
```yaml
document_size: 5MB
comment_count: 1000
verification:
  - load_time: "< 2 seconds"
  - ui_responsive: true
  - search_performance: "< 500ms"
  - save_time: "< 1 second"
  - memory_usage: "< 100MB"
  - scrolling_smooth: true
```

## Verification Automation

### Unit Test Coverage
```yaml
minimum_coverage:
  - statements: 80%
  - branches: 75%
  - functions: 90%
  - lines: 80%

critical_paths:
  - comment_parsing: 95%
  - position_tracking: 95%
  - file_operations: 90%
```

### Integration Test Suite
```yaml
test_matrix:
  - component_pairs: all_adjacent
  - data_flows: all_primary
  - error_scenarios: all_recoverable
  - performance_tests: all_critical_paths

execution:
  - frequency: on_commit
  - timeout: 5_minutes
  - parallel: true
```

### User Acceptance Criteria
```yaml
usability:
  - learn_basics: "< 5 minutes"
  - common_tasks: "< 3 keystrokes"
  - error_recovery: "always_possible"
  - help_available: "context_sensitive"

reliability:
  - data_loss: "never"
  - crash_rate: "< 0.1%"
  - recovery_time: "< 10 seconds"

performance:
  - perceived_speed: "instant"
  - actual_latency: "< 100ms"
  - resource_usage: "reasonable"
```

## Monitoring and Metrics

### Runtime Verification
```yaml
health_checks:
  - memory_usage
  - file_handles
  - thread_count
  - response_times

performance_metrics:
  - parse_duration
  - save_duration
  - search_latency
  - llm_response_time

error_tracking:
  - parse_failures
  - save_errors
  - position_conflicts
  - llm_timeouts

usage_analytics:
  - commands_used
  - comments_created
  - llm_requests
  - search_queries
```

### Quality Gates
```yaml
release_criteria:
  - all_tests_passing: true
  - performance_benchmarks_met: true
  - memory_leaks: none
  - security_scan: passed
  - documentation_complete: true
```