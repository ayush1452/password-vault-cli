# Planned Improvements for Vault CLI

This document outlines the planned improvements and enhancements for the Vault CLI application.

## List Command Enhancements

### 1. Improved Output Formatting
- [ ] **Boxed Layout**
  - Add Unicode box-drawing characters for better visual structure
  - Create consistent borders around the output
  - Improve alignment and padding
  
  **Example:**
  ```
  ┌─────────────────────────────────────────────────┐
  │                  MY VAULT (5)                   │
  ├───────────┬────────────┬────────────────┬────────────┐
  │ Name      │ Username   │ Tags           │ Updated    │
  ├───────────┼────────────┼────────────────┼────────────┤
  │ github    │ user1      │ work, git      │ 2025-11-07 │
  │ aws-prod  │ aws-admin  │ work, aws, prod│ 2025-11-06 │
  │ netflix   │ streamer   │ personal       │ 2025-11-05 │
  └───────────┴────────────┴────────────────┴────────────┘
  ```

- [ ] **Color Support**
  - Add color-coded output for better readability
  - Support for both light and dark terminal themes
  - Configurable color schemes

  **Example with colors:**
  ```
  [1;34m┌─────────────────────────────────────────────────┐[0m
  [1;34m│[0m [1;36mMY VAULT [1;37m(5)[0m                               [1;34m│[0m
  [1;34m├───────────┬────────────┬────────────────┬────────────┤[0m
  [1;34m│[0m [1;33mName     [0m[1;34m│[0m [1;33mUsername  [0m[1;34m│[0m [1;33mTags          [0m[1;34m│[0m [1;33mUpdated   [0m[1;34m│[0m
  [1;34m├───────────┼────────────┼────────────────┼────────────┤[0m
  [1;34m│[0m github    [1;34m│[0m user1     [1;34m│[0m [32mwork[0m, [32mgit[0m      [1;34m│[0m 2025-11-07 [1;34m│[0m
  [1;34m│[0m aws-prod  [1;34m│[0m aws-admin [1;34m│[0m [32mwork[0m, [33maws[0m, [31mprod[0m[1;34m│[0m 2025-11-06 [1;34m│[0m
  [1;34m│[0m netflix   [1;34m│[0m streamer  [1;34m│[0m [36mpersonal[0m       [1;34m│[0m 2025-11-05 [1;34m│[0m
  [1;34m└───────────┴────────────┴────────────────┴────────────┘[0m
  ```
  
  **Color Legend:**
  - [1;36mCyan[0m: Headers
  - [1;33mYellow[0m: Column names
  - [32mGreen[0m: Work-related tags
  - [33mYellow[0m: AWS-related tags
  - [31mRed[0m: Production tags
  - [36mCyan[0m: Personal tags

- [ ] **Customizable Columns**
  - Allow users to choose which columns to display
  - Support for custom column widths
  - Option to reorder columns

  **Example with Custom Columns:**
  ```bash
  # Show only name and URL columns
  vault list --columns name,url
  
  # Example output:
  ┌───────────┬─────────────────────────┐
  │ Name      │ URL                     │
  ├───────────┼─────────────────────────┤
  │ github    │ https://github.com      │
  │ aws-prod  │ https://aws.amazon.com  │
  └───────────┴─────────────────────────┘
  
  # Show with custom widths
  vault list --columns name:15,username:20,tags:30
  ```

### 2. Enhanced Filtering
- [ ] **Advanced Search**
  - Support for regular expressions in search
  - Case-insensitive search option
  - Search in specific fields (name, username, URL, notes)

  **Search Examples:**
  ```bash
  # Basic search (searches in name, username, URL, notes)
  vault list --search "github"
  
  # Search in specific field
  vault list --search "username:admin"
  
  # Regex search
  vault list --search "name:/aws-.*/"
  
  # Multiple conditions (AND)
  vault list --search "work aws"
  
  # Exclude terms
  vault list --search "work -prod"
  ```

- [ ] **Tag Management**
  - Nested tags (e.g., `work/git`, `personal/social`)
  - Tag autocompletion
  - Bulk tag operations

  **Tag Examples:**
  ```bash
  # List entries with nested tags
  vault list --tags work/git
  
  # Multiple tag filters (OR)
  vault list --tags work,personal
  
  # Tag autocompletion
  vault list --tags work/<TAB>
  # Suggests: work/git, work/aws, work/vpn
  
  # Bulk add tag
  vault tag add work/aws --search "name:aws"
  ```

### 3. Sorting and Pagination
- [ ] **Sorting Options**
  - Sort by any column (name, username, last updated, etc.)
  - Reverse sort option
  - Natural sort order for alphanumeric values

  **Sorting Examples:**
  ```bash
  # Sort by name (alphabetical)
  vault list --sort name
  
  # Sort by last updated (newest first)
  vault list --sort -updated
  
  # Sort by tag count
  vault list --sort tags
  
  # Sort by multiple columns
  vault list --sort "type,name"
  ```

- [ ] **Pagination**
  - Limit number of results per page
  - Interactive page navigation
  - Configurable page size

  **Pagination Examples:**
  ```bash
  # Show first 10 entries
  vault list --limit 10
  
  # Show next page
  vault list --page 2
  
  # Interactive mode (with vim-like navigation)
  vault list --interactive
  
  # Output with pagination info
  ┌───────────────────────────────────────────┐
  │ Showing 11-20 of 42 entries (Page 2/5)    │
  └───────────────────────────────────────────┘
  ```

### 4. Export and Integration
- [ ] **Export Formats**
  - CSV export
  - Markdown tables
  - HTML output

- [ ] **Integration**
  - Pipe output to other commands (e.g., `grep`, `jq`)
  - Support for common output formats (JSONL, YAML)

## General Improvements

### Performance
- [ ] Optimize list operations for large vaults
- [ ] Add caching for frequently accessed entries
- [ ] Lazy loading of entry details

### User Experience
- [ ] Interactive mode with keyboard navigation
- [ ] Progress indicators for long operations
- [ ] Better error messages and help text

### Testing
- [ ] Add more test cases for edge cases
- [ ] Performance benchmarking
- [ ] Cross-platform testing

## Implementation Plan

### Phase 1: Core Improvements
1. Implement boxed layout and basic color support
2. Add column customization
3. Implement basic sorting

### Phase 2: Advanced Features
1. Add advanced search capabilities
2. Implement tag management improvements
3. Add pagination

### Phase 3: Polish and Optimization
1. Performance optimizations
2. Additional export formats
3. Enhanced error handling

## Contributing

Contributions are welcome! Please open an issue to discuss any improvements or new features before submitting a pull request.

## License

[Specify your license here]

---
Last Updated: 2025-11-07
