---
title: "FIX Protocol: The Language of Electronic Trading"
date: 2025-01-20
draft: true
description: "A deep dive into the Financial Information eXchange protocol: message structure, session management, and C++ implementation."
---

If you've ever wondered how trading systems communicate, the answer is usually FIX.

The Financial Information eXchange (FIX) protocol is the lingua franca of electronic trading. Every major exchange, broker, and trading platform speaks it. When you place an order through your broker, there's a good chance FIX messages are flowing somewhere in the chain.

This article explains what FIX is, how it works, and how to implement it in C++.

## A Brief History

In 1992, Fidelity Investments had a problem. Their traders were communicating with Salomon Brothers over the phone. Orders got lost. Information went to the wrong trader. When people hung up, context disappeared.

Robert Lamoureux and Chris Morstatt at Fidelity created FIX to solve this. Instead of voice, they wanted machine-readable data.

```
1992: FIX 2.7 - Initial release (Fidelity + Salomon Brothers)
1995: FIX 4.0 - First public version
1998: FIX 4.2 - Most widely adopted version
2001: FIX 4.3 - Added fixed income support
2003: FIX 4.4 - Enhanced market data
2006: FIX 5.0 - Split session/application layers
2024: FIX still going strong
```

Today, FIX is managed by FIX Trading Community, a non-profit with over 300 member firms. It's an open standard—no licensing fees, no vendor lock-in.

## Why FIX Matters

FIX dominates because it solves real problems:

**Standardization**: Before FIX, every broker had a proprietary protocol. Connecting to 10 brokers meant maintaining 10 different integrations. With FIX, you implement once and connect to everyone.

**Reliability**: FIX includes sequence numbers, heartbeats, and message recovery. If a connection drops, both sides know exactly which messages need to be resent.

**Completeness**: FIX covers the entire trade lifecycle:
- Pre-trade: quotes, market data, indications of interest
- Trade: orders, executions, cancellations
- Post-trade: allocations, confirmations, settlement

**Performance**: Despite being text-based, FIX is fast enough for most trading. For ultra-low latency, there's FAST (FIX Adapted for STreaming) and SBE (Simple Binary Encoding).

## Protocol Architecture

FIX operates on two layers:

```
┌─────────────────────────────────────────┐
│           Application Layer             │
│  (Orders, Executions, Market Data)      │
├─────────────────────────────────────────┤
│            Session Layer                │
│  (Logon, Heartbeat, Sequence Numbers)   │
├─────────────────────────────────────────┤
│               TCP/IP                    │
└─────────────────────────────────────────┘
```

**Session Layer**: Handles connection management, authentication, message sequencing, and recovery. This is what keeps the conversation reliable.

**Application Layer**: The actual business messages—orders, executions, market data. This is what traders care about.

Starting with FIX 5.0, these layers were formally separated into FIXT (session) and FIX (application), allowing different application versions over the same session protocol.

## Message Structure

Every FIX message follows the same pattern:

```
Header + Body + Trailer
```

Fields are `tag=value` pairs separated by the SOH character (ASCII 0x01, often shown as `|` or `^`).

### A Real Message

Here's a New Order Single (buying 7000 shares of MSFT):

```
8=FIX.4.4|9=148|35=D|34=1080|49=TESTBUY1|52=20180920-18:14:19.508|
56=TESTSELL1|11=636730640278898634|15=USD|21=2|38=7000|40=1|54=1|
55=MSFT|60=20180920-18:14:19.492|10=092|
```

Let's break it down:

### Header Fields (Required)

| Tag | Name | Value | Meaning |
|-----|------|-------|---------|
| 8 | BeginString | FIX.4.4 | Protocol version |
| 9 | BodyLength | 148 | Message body size in bytes |
| 35 | MsgType | D | New Order Single |
| 34 | MsgSeqNum | 1080 | Sequence number |
| 49 | SenderCompID | TESTBUY1 | Who's sending |
| 52 | SendingTime | 20180920-18:14:19.508 | When it was sent |
| 56 | TargetCompID | TESTSELL1 | Who should receive |

### Body Fields (Message-Specific)

| Tag | Name | Value | Meaning |
|-----|------|-------|---------|
| 11 | ClOrdID | 636730640278898634 | Client's order ID |
| 15 | Currency | USD | Currency |
| 21 | HandlInst | 2 | Automated execution |
| 38 | OrderQty | 7000 | Quantity |
| 40 | OrdType | 1 | Market order |
| 54 | Side | 1 | Buy |
| 55 | Symbol | MSFT | Instrument |
| 60 | TransactTime | 20180920-18:14:19.492 | Transaction time |

### Trailer (Required)

| Tag | Name | Value | Meaning |
|-----|------|-------|---------|
| 10 | CheckSum | 092 | Simple checksum for validation |

The checksum is calculated as: `sum of all bytes % 256`, formatted as 3 digits.

## Key Message Types

FIX defines dozens of message types. Here are the essential ones:

### Session Messages

| MsgType | Name | Purpose |
|---------|------|---------|
| A | Logon | Initiate session |
| 5 | Logout | Terminate session |
| 0 | Heartbeat | Keep-alive |
| 1 | Test Request | Verify connection |
| 2 | Resend Request | Request missed messages |
| 4 | Sequence Reset | Reset sequence numbers |
| 3 | Reject | Session-level rejection |

### Application Messages

| MsgType | Name | Purpose |
|---------|------|---------|
| D | New Order Single | Submit a new order |
| F | Order Cancel Request | Cancel an order |
| G | Order Cancel/Replace | Modify an order |
| 8 | Execution Report | Order status/fill |
| 9 | Order Cancel Reject | Cancel failed |
| V | Market Data Request | Subscribe to quotes |
| W | Market Data Snapshot | Full book snapshot |
| X | Market Data Incremental | Book updates |

## Session Management

FIX sessions are stateful. Both sides maintain sequence numbers and can recover from failures.

### Session Lifecycle

```
┌──────────┐     Logon (A)      ┌──────────┐
│ Initiator │ ──────────────────▶│ Acceptor │
│          │◀────────────────── │          │
└──────────┘     Logon (A)      └──────────┘
      │                               │
      │  ◀── Application Messages ──▶ │
      │  ◀──    Heartbeats (0)    ──▶ │
      │                               │
      │         Logout (5)            │
      │ ──────────────────────────────▶
      │◀──────────────────────────────
      │         Logout (5)            │
```

### Sequence Numbers

Every message has a sequence number (tag 34). Both sides track:
- **NextNumOut**: Next sequence number to send
- **NextNumIn**: Next sequence number expected

If you receive MsgSeqNum 100 but expected 95, there's a gap. You send a Resend Request for messages 95-99.

```
Expected: 95
Received: 100  (Gap detected!)

     ┌──────────────────────────────────────┐
     │ Resend Request: BeginSeqNo=95        │
     │                 EndSeqNo=99          │
     └──────────────────────────────────────┘
```

### Heartbeats

During idle periods, both sides send Heartbeat messages at a configured interval (typically 30 seconds). This:
1. Proves the connection is alive
2. Allows gap detection even during quiet periods

If no Heartbeat arrives within `HeartBtInt + reasonable_delta`, the connection is considered dead.

### Message Recovery

When reconnecting after a failure:

1. **Initiator** sends Logon with its NextNumOut
2. **Acceptor** checks against its NextNumIn
3. If there's a gap, Acceptor sends Resend Request
4. Initiator retransmits missing messages with PossDupFlag=Y
5. Normal messaging resumes

This is why FIX can survive network failures without losing orders.

## Important Tags Reference

### Order Tags

| Tag | Name | Description |
|-----|------|-------------|
| 1 | Account | Trading account |
| 11 | ClOrdID | Client order ID (your reference) |
| 37 | OrderID | Exchange order ID (their reference) |
| 38 | OrderQty | Order quantity |
| 40 | OrdType | 1=Market, 2=Limit, 3=Stop, etc. |
| 44 | Price | Limit price |
| 54 | Side | 1=Buy, 2=Sell, 5=Short |
| 55 | Symbol | Instrument symbol |
| 59 | TimeInForce | 0=Day, 1=GTC, 3=IOC, 4=FOK |

### Execution Report Tags

| Tag | Name | Description |
|-----|------|-------------|
| 6 | AvgPx | Average fill price |
| 14 | CumQty | Cumulative filled quantity |
| 17 | ExecID | Execution ID |
| 31 | LastPx | Last fill price |
| 32 | LastQty | Last fill quantity |
| 39 | OrdStatus | 0=New, 1=PartialFill, 2=Filled, 4=Canceled, 8=Rejected |
| 150 | ExecType | What happened (0=New, F=Trade, 4=Canceled, etc.) |
| 151 | LeavesQty | Remaining quantity |

### Execution Report Example

```
8=FIX.4.4|9=289|35=8|34=1090|49=TESTSELL1|52=20180920-18:23:53.671|
56=TESTBUY1|6=113.35|11=636730640278898634|14=3500|15=USD|17=20636730646335310000|
21=2|31=113.35|32=3500|37=20636730646335310000|38=7000|39=1|40=1|54=1|55=MSFT|
60=20180920-18:23:53.531|150=F|151=3500|10=151|
```

This says: Your 7000-share MSFT order (tag 11) is partially filled (tag 39=1). 3500 shares executed at $113.35 (tags 32, 31). 3500 shares remain (tag 151).

## User-Defined Tags

Tags 5000-9999 (and 20000-39999 in newer versions) are reserved for custom fields. Trading partners can agree on their own meanings:

```
8=FIX.4.4|...|5001=INTERNAL_STRATEGY_7|5002=URGENT|...
```

This allows firms to extend FIX without breaking compatibility.

## Repeating Groups

Some data needs to repeat—like multiple legs of a spread order or multiple parties to a trade.

Repeating groups start with a count field, followed by the repeated fields:

```
453=2|                    ← Number of parties (NoPartyIDs)
  448=BROKER1|447=D|452=1|  ← Party 1
  448=TRADER1|447=D|452=11| ← Party 2
```

Rules:
1. Count field comes first
2. Fields within a group must maintain consistent order
3. The first field of each instance identifies the group boundary

## FIXML: The XML Alternative

FIX also has an XML format:

```xml
<FIXML>
  <Order ClOrdID="12345" Side="1" OrdTyp="2" Px="100.50">
    <Instrmt Sym="MSFT"/>
    <OrdQty Qty="1000"/>
  </Order>
</FIXML>
```

FIXML is:
- More verbose (larger messages)
- Easier to validate (XML Schema)
- Better for complex, nested structures
- Popular in post-trade processing

Tag=value remains dominant for real-time trading due to size and parsing speed.

## C++ Implementation

There are several approaches to implementing FIX in C++:

### 1. QuickFIX (Full Framework)

[QuickFIX](https://github.com/quickfix/quickfix) is the most popular open-source FIX engine. It handles everything: sessions, threading, persistence.

```cpp
// QuickFIX: Creating a New Order Single
#include "quickfix/fix44/NewOrderSingle.h"

void sendOrder() {
    FIX44::NewOrderSingle order(
        FIX::ClOrdID("ORDER123"),
        FIX::Side(FIX::Side_BUY),
        FIX::TransactTime(),
        FIX::OrdType(FIX::OrdType_LIMIT)
    );

    order.set(FIX::Symbol("MSFT"));
    order.set(FIX::OrderQty(1000));
    order.set(FIX::Price(150.25));
    order.set(FIX::TimeInForce(FIX::TimeInForce_DAY));

    FIX::Session::sendToTarget(order, sessionID);
}
```

QuickFIX application callback:

```cpp
class MyApplication : public FIX::Application {
public:
    void onCreate(const FIX::SessionID&) override {}

    void onLogon(const FIX::SessionID& sessionID) override {
        std::cout << "Logged on: " << sessionID << std::endl;
    }

    void onLogout(const FIX::SessionID& sessionID) override {
        std::cout << "Logged out: " << sessionID << std::endl;
    }

    void toAdmin(FIX::Message&, const FIX::SessionID&) override {}
    void fromAdmin(const FIX::Message&, const FIX::SessionID&) override {}

    void toApp(FIX::Message&, const FIX::SessionID&) override {}

    void fromApp(const FIX::Message& message,
                 const FIX::SessionID& sessionID) override {
        crack(message, sessionID);
    }

    // Handle Execution Reports
    void onMessage(const FIX44::ExecutionReport& report,
                   const FIX::SessionID&) {
        FIX::ExecType execType;
        FIX::OrdStatus ordStatus;
        FIX::ClOrdID clOrdID;

        report.get(execType);
        report.get(ordStatus);
        report.get(clOrdID);

        std::cout << "Execution: " << clOrdID
                  << " Status: " << ordStatus << std::endl;
    }
};
```

### 2. hffix (High-Frequency Parser)

[hffix](https://jamesdbrock.github.io/hffix/) is a header-only library for low-latency applications. No heap allocations, no object overhead.

```cpp
#include <hffix.hpp>

// Writing a FIX message
void writeLogon(char* buffer, size_t bufferSize) {
    hffix::message_writer logon(buffer, buffer + bufferSize);

    logon.push_back_header("FIX.4.2");
    logon.push_back_string(hffix::tag::MsgType, "A");
    logon.push_back_string(hffix::tag::SenderCompID, "CLIENT1");
    logon.push_back_string(hffix::tag::TargetCompID, "BROKER1");
    logon.push_back_int(hffix::tag::MsgSeqNum, 1);
    logon.push_back_timestamp(hffix::tag::SendingTime,
        std::chrono::system_clock::now());
    logon.push_back_int(hffix::tag::EncryptMethod, 0);
    logon.push_back_int(hffix::tag::HeartBtInt, 30);
    logon.push_back_trailer();

    // Message is now in buffer, ready to send
    size_t messageLength = logon.message_end() - buffer;
}

// Reading a FIX message
void readMessage(const char* buffer, size_t length) {
    hffix::message_reader reader(buffer, buffer + length);

    if (!reader.is_complete()) {
        // Need more data
        return;
    }

    if (!reader.is_valid()) {
        // Checksum failed
        return;
    }

    // Get message type
    auto msgType = reader.find_with_hint(hffix::tag::MsgType);
    if (msgType != reader.end()) {
        if (msgType->value() == "8") {
            // Execution Report
            processExecutionReport(reader);
        }
    }
}

void processExecutionReport(hffix::message_reader& reader) {
    for (auto field = reader.begin(); field != reader.end(); ++field) {
        switch (field->tag()) {
            case hffix::tag::ClOrdID:
                std::cout << "ClOrdID: " << field->value() << std::endl;
                break;
            case hffix::tag::OrdStatus:
                std::cout << "Status: " << field->value() << std::endl;
                break;
            case hffix::tag::CumQty:
                std::cout << "Filled: " << field->value().as_int<int>() << std::endl;
                break;
            case hffix::tag::LeavesQty:
                std::cout << "Remaining: " << field->value().as_int<int>() << std::endl;
                break;
        }
    }
}
```

### 3. Custom Parser (Minimal)

For maximum control, you can parse FIX manually:

```cpp
#include <string>
#include <string_view>
#include <unordered_map>
#include <charconv>

constexpr char SOH = '\x01';

class FIXMessage {
public:
    void parse(std::string_view data) {
        fields_.clear();

        size_t pos = 0;
        while (pos < data.size()) {
            // Find '='
            size_t eqPos = data.find('=', pos);
            if (eqPos == std::string_view::npos) break;

            // Find SOH
            size_t sohPos = data.find(SOH, eqPos);
            if (sohPos == std::string_view::npos) {
                sohPos = data.size();
            }

            // Extract tag and value
            int tag = 0;
            std::from_chars(data.data() + pos, data.data() + eqPos, tag);

            std::string_view value(data.data() + eqPos + 1,
                                   sohPos - eqPos - 1);
            fields_[tag] = std::string(value);

            pos = sohPos + 1;
        }
    }

    std::string_view get(int tag) const {
        auto it = fields_.find(tag);
        if (it != fields_.end()) {
            return it->second;
        }
        return {};
    }

    int getInt(int tag) const {
        auto val = get(tag);
        int result = 0;
        std::from_chars(val.data(), val.data() + val.size(), result);
        return result;
    }

    bool has(int tag) const {
        return fields_.find(tag) != fields_.end();
    }

private:
    std::unordered_map<int, std::string> fields_;
};

// Usage
int main() {
    std::string raw = "8=FIX.4.4\x01" "9=100\x01" "35=D\x01"
                      "49=SENDER\x01" "56=TARGET\x01"
                      "11=ORDER123\x01" "55=MSFT\x01"
                      "38=1000\x01" "10=123\x01";

    FIXMessage msg;
    msg.parse(raw);

    std::cout << "MsgType: " << msg.get(35) << std::endl;
    std::cout << "Symbol: " << msg.get(55) << std::endl;
    std::cout << "Quantity: " << msg.getInt(38) << std::endl;
}
```

## Complete Example: Order Flow

Here's a realistic order flow implementation:

```cpp
// fix_client.cpp
// A simple FIX 4.4 trading client using hffix

#include <iostream>
#include <cstring>
#include <chrono>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>

// Include hffix (header-only)
// #include <hffix.hpp>

// For this example, we'll use a simplified implementation
constexpr char SOH = '\x01';

class FIXClient {
public:
    FIXClient(const std::string& senderCompID,
              const std::string& targetCompID)
        : senderCompID_(senderCompID)
        , targetCompID_(targetCompID)
        , seqNum_(1)
        , sockfd_(-1)
    {}

    bool connect(const std::string& host, int port) {
        sockfd_ = socket(AF_INET, SOCK_STREAM, 0);
        if (sockfd_ < 0) return false;

        sockaddr_in addr{};
        addr.sin_family = AF_INET;
        addr.sin_port = htons(port);
        inet_pton(AF_INET, host.c_str(), &addr.sin_addr);

        if (::connect(sockfd_, (sockaddr*)&addr, sizeof(addr)) < 0) {
            close(sockfd_);
            sockfd_ = -1;
            return false;
        }

        return true;
    }

    void disconnect() {
        if (sockfd_ >= 0) {
            sendLogout();
            close(sockfd_);
            sockfd_ = -1;
        }
    }

    bool logon(int heartbeatInterval = 30) {
        std::string body;
        addField(body, 98, "0");                      // EncryptMethod = None
        addField(body, 108, std::to_string(heartbeatInterval)); // HeartBtInt

        std::string msg = buildMessage("A", body);
        return sendRaw(msg);
    }

    bool sendLogout() {
        std::string body;
        std::string msg = buildMessage("5", body);
        return sendRaw(msg);
    }

    bool sendNewOrderSingle(const std::string& clOrdID,
                            const std::string& symbol,
                            char side,        // '1' = Buy, '2' = Sell
                            int quantity,
                            char ordType,     // '1' = Market, '2' = Limit
                            double price = 0.0) {
        std::string body;

        addField(body, 11, clOrdID);                  // ClOrdID
        addField(body, 21, "1");                      // HandlInst = Automated
        addField(body, 55, symbol);                   // Symbol
        addField(body, 54, std::string(1, side));     // Side
        addField(body, 60, timestamp());              // TransactTime
        addField(body, 38, std::to_string(quantity)); // OrderQty
        addField(body, 40, std::string(1, ordType));  // OrdType

        if (ordType == '2' && price > 0) {            // Limit order
            addField(body, 44, formatPrice(price));   // Price
        }

        std::string msg = buildMessage("D", body);
        return sendRaw(msg);
    }

    bool sendCancelRequest(const std::string& origClOrdID,
                           const std::string& clOrdID,
                           const std::string& symbol,
                           char side) {
        std::string body;

        addField(body, 41, origClOrdID);              // OrigClOrdID
        addField(body, 11, clOrdID);                  // ClOrdID
        addField(body, 55, symbol);                   // Symbol
        addField(body, 54, std::string(1, side));     // Side
        addField(body, 60, timestamp());              // TransactTime

        std::string msg = buildMessage("F", body);
        return sendRaw(msg);
    }

    std::string receive() {
        char buffer[4096];
        ssize_t n = recv(sockfd_, buffer, sizeof(buffer) - 1, 0);
        if (n <= 0) return "";
        buffer[n] = '\0';
        return std::string(buffer, n);
    }

private:
    std::string buildMessage(const std::string& msgType,
                             const std::string& body) {
        // Build body with standard fields
        std::string fullBody;
        addField(fullBody, 35, msgType);              // MsgType
        addField(fullBody, 49, senderCompID_);        // SenderCompID
        addField(fullBody, 56, targetCompID_);        // TargetCompID
        addField(fullBody, 34, std::to_string(seqNum_++)); // MsgSeqNum
        addField(fullBody, 52, timestamp());          // SendingTime
        fullBody += body;

        // Build header
        std::string header;
        addField(header, 8, "FIX.4.4");               // BeginString
        addField(header, 9, std::to_string(fullBody.size())); // BodyLength

        // Combine and add checksum
        std::string msg = header + fullBody;
        int checksum = 0;
        for (char c : msg) checksum += static_cast<unsigned char>(c);
        checksum %= 256;

        char checksumStr[4];
        snprintf(checksumStr, sizeof(checksumStr), "%03d", checksum);
        addField(msg, 10, checksumStr);               // CheckSum

        return msg;
    }

    void addField(std::string& msg, int tag, const std::string& value) {
        msg += std::to_string(tag) + "=" + value + SOH;
    }

    std::string timestamp() {
        auto now = std::chrono::system_clock::now();
        auto time = std::chrono::system_clock::to_time_t(now);
        auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(
            now.time_since_epoch()) % 1000;

        char buf[32];
        std::strftime(buf, sizeof(buf), "%Y%m%d-%H:%M:%S", std::gmtime(&time));

        char result[64];
        snprintf(result, sizeof(result), "%s.%03d", buf, (int)ms.count());
        return result;
    }

    std::string formatPrice(double price) {
        char buf[32];
        snprintf(buf, sizeof(buf), "%.2f", price);
        return buf;
    }

    bool sendRaw(const std::string& msg) {
        return send(sockfd_, msg.c_str(), msg.size(), 0) == (ssize_t)msg.size();
    }

    std::string senderCompID_;
    std::string targetCompID_;
    int seqNum_;
    int sockfd_;
};

// Example usage
int main() {
    FIXClient client("MYCLIENT", "BROKER");

    if (!client.connect("127.0.0.1", 9876)) {
        std::cerr << "Failed to connect" << std::endl;
        return 1;
    }

    // Logon
    client.logon(30);
    std::cout << "Sent Logon" << std::endl;

    // Wait for logon response
    std::string response = client.receive();
    std::cout << "Received: " << response << std::endl;

    // Send a limit order
    client.sendNewOrderSingle(
        "ORD001",    // ClOrdID
        "MSFT",      // Symbol
        '1',         // Buy
        100,         // 100 shares
        '2',         // Limit order
        150.50       // Price
    );
    std::cout << "Sent New Order" << std::endl;

    // Wait for execution report
    response = client.receive();
    std::cout << "Received: " << response << std::endl;

    // Disconnect
    client.disconnect();

    return 0;
}
```

## Performance Considerations

### Parsing Efficiency

FIX parsing can be a bottleneck in high-frequency trading:

```cpp
// Slow: String operations, map lookups
std::map<int, std::string> fields;
// Parse and store all fields...
auto symbol = fields[55];  // Map lookup

// Fast: Scan once, process inline
for (auto field = reader.begin(); field != reader.end(); ++field) {
    if (field->tag() == 55) {
        // Process symbol immediately
    }
}
```

### Memory Allocation

Avoid heap allocations in the hot path:

```cpp
// Slow: Allocates on every message
std::string value = field.value_string();

// Fast: Use string_view, no allocation
std::string_view value = field.value();
```

### Checksum Calculation

The FIX checksum is simple but can be optimized:

```cpp
// Naive
int checksum = 0;
for (char c : message) {
    checksum += (unsigned char)c;
}
checksum %= 256;

// SIMD-friendly (compiler may auto-vectorize)
uint32_t checksum = 0;
const char* p = message.data();
const char* end = p + message.size();

// Process 4 bytes at a time
while (p + 4 <= end) {
    checksum += (unsigned char)p[0];
    checksum += (unsigned char)p[1];
    checksum += (unsigned char)p[2];
    checksum += (unsigned char)p[3];
    p += 4;
}
while (p < end) {
    checksum += (unsigned char)*p++;
}
checksum %= 256;
```

## Testing and Debugging

### FIX Log Format

For readability, replace SOH with `|`:

```bash
# Convert FIX logs for viewing
sed 's/\x01/|/g' fix.log
```

### FIX Analyzers

- **FIX Antenna Log Viewer**: GUI for analyzing FIX logs
- **Wireshark**: Has FIX protocol dissector
- **fixspec.com**: Online FIX message decoder

### Simulators

For testing without a real exchange:

- **QuickFIX Executor**: Simple order matching engine
- **fix-simple-server**: Minimal FIX acceptor for testing
- **FIXimulator**: Full-featured exchange simulator

## Beyond FIX: Related Protocols

FIX has spawned several related standards:

**FAST** (FIX Adapted for STreaming): Binary encoding for market data. 10-20x smaller than FIX.

**SBE** (Simple Binary Encoding): Modern binary format used by CME, Binance. Even faster than FAST.

**FpML** (Financial products Markup Language): XML-based protocol for OTC derivatives.

## Summary

FIX is the backbone of electronic trading:

- **Message Format**: `tag=value` pairs separated by SOH
- **Session Layer**: Sequence numbers, heartbeats, recovery
- **Application Layer**: Orders, executions, market data
- **Implementation**: QuickFIX for full features, hffix for low latency

The protocol is old (1992) but far from obsolete. Understanding FIX is essential for anyone working in trading technology.

For ultra-low latency, consider FAST or SBE. For everything else, FIX remains the standard.

## References

### Official Documentation

- [FIX Trading Community](https://www.fixtrading.org/) - Official FIX specifications
- [FIX Beginners Guide](https://www.fixtrading.org/beginners-resources/) - Official learning resources
- [OnixS FIX Dictionary](https://www.onixs.biz/fix-dictionary.html) - Complete tag reference
- [FIX Protocol Wikipedia](https://en.wikipedia.org/wiki/Financial_Information_eXchange)

### C++ Libraries (GitHub)

- [QuickFIX](https://github.com/quickfix/quickfix) - Most popular open source FIX engine
- [hffix](https://github.com/jamesdbrock/hffix) - Header-only, zero-allocation high-frequency parser
- [libtrading](https://github.com/libtrading/libtrading) - Ultra low-latency trading connectivity (FIX, ITCH, OUCH)
- [SubZero](https://github.com/simondevenish/SubZero) - Ultra-low-latency FIX/FAST library
- [crocofix](https://github.com/GaryHughes/crocofix) - Modern C++23 FIX implementation

### Other Languages

- [QuickFIX/J](https://github.com/quickfix/quickfixj) - Java FIX engine
- [Philadelphia](https://github.com/paritytrading/philadelphia) - Fast JVM FIX library
- [simplefix](https://github.com/da4089/simplefix) - Simple Python FIX implementation
- [QuickFIX/Go](https://github.com/quickfixgo/quickfix) - Go FIX engine

### Trading Engine Examples

- [FIX-Trading-Engine](https://github.com/datstma/FIX-Trading-Engine) - Automated trading engine example
- [fix-trading-simulator](https://github.com/felipewind/fix-trading-simulator) - Broker/Exchange simulator (QuickFIX/J + Quarkus)
- [pytradesimulator](https://github.com/abhi-g80/pytradesimulator) - Python exchange with FIFO matching
- [UltraLowLatencyFeedHandler](https://github.com/harris2001/UltraLowLatencyFeedHandler) - C++ ITCH/FIX feed handler

### Exchange Simulators

- [FIXimulator](http://fiximulator.org/) - Java sell-side simulator
- [exsim](https://github.com/da4089/exsim) - Basic exchange simulator

### Online Tools

- [FIXSIM Parser](https://www.fixsim.com/fix-parser) - Online message decoder
- [Aprics FIX Parser](https://fix.aprics.net/) - Browser-based, no data sent to server
- [ZagTrader Parser](https://fixparser.zagtrader.com/) - Timeline view, color-coded tags
- [OnixS FIX Analyser](https://www.onixs.biz/fix-analyser.html) - Desktop log analyzer

### Tutorials & Articles

- [FIXSIM Tutorial](https://www.fixsim.com/fix-protocol-tutorial) - Comprehensive FIX tutorial
- [Habr: FIX Protocol (RU)](https://habr.com/ru/companies/iticapital/articles/242789/) - Russian introduction
- [Habr: Trading with FIX (RU)](https://habr.com/ru/articles/503916/) - Practical setup guide
- [Habr: FAST Protocol (RU)](https://habr.com/ru/articles/827330/) - FIX streaming extension
- [FIX Message Samples](https://www.fixsim.com/sample-fix-messages) - Example messages

### Books

- [Building Low Latency Applications with C++](https://github.com/PacktPublishing/Building-Low-Latency-Applications-with-CPP) - HFT systems book with code
