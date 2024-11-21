# NGINX Parser

## Project Description

**NGINX Parser** is a tool for analyzing NGINX log files. It simplifies working with logs by providing detailed statistics on requests, response sizes, HTTP codes, and other parameters. The program supports both local files (with wildcard patterns) and remote files via URL. The program processes data in a streaming mode without loading the entire file into memory. The analysis results are presented in convenient formats: **Markdown** or **AsciiDoc**.

The NGINX log format is:  
`$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"`.

---

## Features

1. Counts the total number of requests.
2. Identifies the most frequently requested resources.
3. Counts the most common HTTP response codes.
4. Counts the most common IP addresses.
5. Calculates the average server response size.
6. Computes the 95th percentile of response sizes.
7. Calculates the average number of requests per day.
8. Filters logs by time range (`from` and `to` in ISO8601 format).
9. Supports output in **Markdown** or **AsciiDoc** format.
10. Processes both local files (including patterns) and URLs.
11. Supports filtering logs by specific values.

---

## Installation and Running

1. Clone the repository:
   ```bash
   git clone git@github.com:LLIEPJIOK/nginxparser.git
   ```
2. Navigate to the repository folder:
   ```bash
   cd nginxparser
   ```
3. Run the program:
   ```bash
   go run cmd/parser/main.go -p <path_to_logs> <additional_flags>
   ```

---

## Example

### Command

```bash
go run cmd/parser/main.go -p logs/*/* -filter-field Status -filter-value 404 -fmt adoc -o result.txt
```

### Output

```adoc
==== General Information

[options="header"]
|===
| Metric                 | Value
| Files                  | logs/log1/logs.txt, logs/log2/logs.txt
| Number of requests     | 552
| Average response size  | 74
| 95th percentile of response size | 117
| Average requests per day | 552
|===

==== Requested Resources

[options="header"]
|===
| Resource                             | Count
| `/composite.svg`                     | 4
| `/24/7-encoding/systematic-grid-enabled.png` | 2
| `/24/7.js`                           | 2
|===

==== Response Codes

[options="header"]
|===
| Code | Name          | Count
| 404  | Not Found     | 552
|===

==== Requesting Addresses

[options="header"]
|===
| Address         | Count
| 1.223.212.18    | 2
| 10.218.230.66   | 2
| 10.88.70.207    | 2
|===
```

---

## Testing

1. **Log Reading**:

   - Verify reading of local files and URLs.
   - Test wildcard patterns for local file paths.

2. **Log Parsing**:

   - Validate correct parsing of NGINX log lines.

3. **Filtering**:

   - Test filtering records by time range.
   - Test filtering by specific fields (e.g., `GET` method or `Mozilla` user agent).

4. **Statistics Calculation**:

   - Verify the accuracy of all computed metrics.

5. **Output**:
   - Validate that the output meets Markdown and AsciiDoc format requirements.
