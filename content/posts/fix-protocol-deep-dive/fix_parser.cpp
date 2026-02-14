/**
 * FIX Protocol Parser and Message Builder
 *
 * A complete example demonstrating:
 * - FIX message parsing
 * - FIX message building
 * - Checksum calculation
 * - Common message types (Logon, NewOrderSingle, ExecutionReport)
 *
 * Compile: g++ -std=c++17 -O2 fix_parser.cpp -o fix_parser
 * Run: ./fix_parser
 */

#include <iostream>
#include <string>
#include <string_view>
#include <unordered_map>
#include <vector>
#include <charconv>
#include <chrono>
#include <iomanip>
#include <sstream>
#include <cstring>
#include <optional>

// ASCII SOH (Start of Header) - FIX field delimiter
constexpr char SOH = '\x01';

// Common FIX tags
namespace Tag {
    constexpr int BeginString    = 8;
    constexpr int BodyLength     = 9;
    constexpr int CheckSum       = 10;
    constexpr int ClOrdID        = 11;
    constexpr int CumQty         = 14;
    constexpr int ExecID         = 17;
    constexpr int HandlInst      = 21;
    constexpr int LastPx         = 31;
    constexpr int LastQty        = 32;
    constexpr int MsgSeqNum      = 34;
    constexpr int MsgType        = 35;
    constexpr int OrderID        = 37;
    constexpr int OrderQty       = 38;
    constexpr int OrdStatus      = 39;
    constexpr int OrdType        = 40;
    constexpr int OrigClOrdID    = 41;
    constexpr int Price          = 44;
    constexpr int SenderCompID   = 49;
    constexpr int SendingTime    = 52;
    constexpr int Side           = 54;
    constexpr int Symbol         = 55;
    constexpr int TargetCompID   = 56;
    constexpr int Text           = 58;
    constexpr int TimeInForce    = 59;
    constexpr int TransactTime   = 60;
    constexpr int EncryptMethod  = 98;
    constexpr int HeartBtInt     = 108;
    constexpr int ExecType       = 150;
    constexpr int LeavesQty      = 151;
}

// Message types
namespace MsgType {
    constexpr char Heartbeat         = '0';
    constexpr char TestRequest       = '1';
    constexpr char ResendRequest     = '2';
    constexpr char Reject            = '3';
    constexpr char SequenceReset     = '4';
    constexpr char Logout            = '5';
    constexpr char ExecutionReport   = '8';
    constexpr char OrderCancelReject = '9';
    constexpr char Logon             = 'A';
    constexpr char NewOrderSingle    = 'D';
    constexpr char OrderCancelReq    = 'F';
    constexpr char OrderReplaceReq   = 'G';
}

// Order Status
namespace OrdStatus {
    constexpr char New           = '0';
    constexpr char PartialFill   = '1';
    constexpr char Filled        = '2';
    constexpr char DoneForDay    = '3';
    constexpr char Canceled      = '4';
    constexpr char Replaced      = '5';
    constexpr char PendingCancel = '6';
    constexpr char Stopped       = '7';
    constexpr char Rejected      = '8';
}

// Execution Type
namespace ExecType {
    constexpr char New           = '0';
    constexpr char PartialFill   = '1';
    constexpr char Fill          = '2';
    constexpr char DoneForDay    = '3';
    constexpr char Canceled      = '4';
    constexpr char Replaced      = '5';
    constexpr char PendingCancel = '6';
    constexpr char Rejected      = '8';
    constexpr char Trade         = 'F';
}

// Side
namespace Side {
    constexpr char Buy  = '1';
    constexpr char Sell = '2';
}

// Order Type
namespace OrdType {
    constexpr char Market = '1';
    constexpr char Limit  = '2';
    constexpr char Stop   = '3';
}

// ============================================================================
// FIX Field
// ============================================================================

struct FIXField {
    int tag;
    std::string value;

    FIXField(int t, std::string_view v) : tag(t), value(v) {}

    int asInt() const {
        int result = 0;
        std::from_chars(value.data(), value.data() + value.size(), result);
        return result;
    }

    double asDouble() const {
        // Simple parsing (production code should handle locales)
        if (value.empty()) return 0.0;
        try {
            return std::stod(value);
        } catch (...) {
            return 0.0;
        }
    }

    char asChar() const {
        return value.empty() ? '\0' : value[0];
    }
};

// ============================================================================
// FIX Message Reader
// ============================================================================

class FIXMessageReader {
public:
    bool parse(std::string_view data) {
        fields_.clear();
        raw_ = data;

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

            // Parse tag
            int tag = 0;
            auto [ptr, ec] = std::from_chars(
                data.data() + pos,
                data.data() + eqPos,
                tag
            );

            if (ec == std::errc()) {
                std::string_view value(
                    data.data() + eqPos + 1,
                    sohPos - eqPos - 1
                );
                fields_.emplace_back(tag, value);
            }

            pos = sohPos + 1;
        }

        return !fields_.empty() && validateChecksum();
    }

    std::optional<FIXField> find(int tag) const {
        for (const auto& field : fields_) {
            if (field.tag == tag) {
                return field;
            }
        }
        return std::nullopt;
    }

    std::string_view get(int tag) const {
        auto field = find(tag);
        return field ? std::string_view(field->value) : std::string_view{};
    }

    int getInt(int tag) const {
        auto field = find(tag);
        return field ? field->asInt() : 0;
    }

    double getDouble(int tag) const {
        auto field = find(tag);
        if (!field) return 0.0;
        if (field->value.empty()) return 0.0;
        return field->asDouble();
    }

    char getChar(int tag) const {
        auto field = find(tag);
        return field ? field->asChar() : '\0';
    }

    char msgType() const {
        return getChar(Tag::MsgType);
    }

    bool isValid() const { return !fields_.empty(); }

    const std::vector<FIXField>& fields() const { return fields_; }

    // Pretty print
    void dump(std::ostream& os) const {
        for (const auto& field : fields_) {
            os << field.tag << "=" << field.value << "|";
        }
        os << std::endl;
    }

private:
    bool validateChecksum() const {
        auto checksumField = find(Tag::CheckSum);
        if (!checksumField) return false;

        // Calculate checksum of everything before CheckSum field
        size_t checksumPos = raw_.rfind("10=");
        if (checksumPos == std::string_view::npos) return false;

        int calculated = 0;
        for (size_t i = 0; i < checksumPos; ++i) {
            calculated += static_cast<unsigned char>(raw_[i]);
        }
        calculated %= 256;

        int expected = checksumField->asInt();
        return calculated == expected;
    }

    std::vector<FIXField> fields_;
    std::string_view raw_;
};

// ============================================================================
// FIX Message Writer
// ============================================================================

class FIXMessageWriter {
public:
    FIXMessageWriter(std::string_view beginString = "FIX.4.4")
        : beginString_(beginString)
    {}

    FIXMessageWriter& setField(int tag, std::string_view value) {
        body_ += std::to_string(tag) + "=" + std::string(value) + SOH;
        return *this;
    }

    FIXMessageWriter& setField(int tag, int value) {
        return setField(tag, std::to_string(value));
    }

    FIXMessageWriter& setField(int tag, double value, int precision = 2) {
        std::ostringstream oss;
        oss << std::fixed << std::setprecision(precision) << value;
        return setField(tag, oss.str());
    }

    FIXMessageWriter& setField(int tag, char value) {
        return setField(tag, std::string(1, value));
    }

    FIXMessageWriter& setMsgType(char msgType) {
        msgType_ = msgType;
        return *this;
    }

    FIXMessageWriter& setSender(std::string_view sender) {
        sender_ = sender;
        return *this;
    }

    FIXMessageWriter& setTarget(std::string_view target) {
        target_ = target;
        return *this;
    }

    FIXMessageWriter& setSeqNum(int seqNum) {
        seqNum_ = seqNum;
        return *this;
    }

    std::string build() {
        // Build the body with standard header fields
        std::string fullBody;
        fullBody += std::to_string(Tag::MsgType) + "=" + msgType_ + SOH;
        fullBody += std::to_string(Tag::SenderCompID) + "=" + sender_ + SOH;
        fullBody += std::to_string(Tag::TargetCompID) + "=" + target_ + SOH;
        fullBody += std::to_string(Tag::MsgSeqNum) + "=" + std::to_string(seqNum_) + SOH;
        fullBody += std::to_string(Tag::SendingTime) + "=" + timestamp() + SOH;
        fullBody += body_;

        // Build header (BeginString + BodyLength)
        std::string header;
        header += std::to_string(Tag::BeginString) + "=" + beginString_ + SOH;
        header += std::to_string(Tag::BodyLength) + "=" + std::to_string(fullBody.size()) + SOH;

        // Combine header and body
        std::string message = header + fullBody;

        // Calculate and append checksum
        int checksum = 0;
        for (unsigned char c : message) {
            checksum += c;
        }
        checksum %= 256;

        char checksumStr[8];
        snprintf(checksumStr, sizeof(checksumStr), "%03d", checksum);
        message += std::to_string(Tag::CheckSum) + "=" + checksumStr + SOH;

        // Clear body for reuse
        body_.clear();

        return message;
    }

private:
    static std::string timestamp() {
        auto now = std::chrono::system_clock::now();
        auto time_t = std::chrono::system_clock::to_time_t(now);
        auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(
            now.time_since_epoch()
        ) % 1000;

        std::tm tm = *std::gmtime(&time_t);
        char buf[32];
        std::strftime(buf, sizeof(buf), "%Y%m%d-%H:%M:%S", &tm);

        std::ostringstream oss;
        oss << buf << "." << std::setfill('0') << std::setw(3) << ms.count();
        return oss.str();
    }

    std::string beginString_;
    std::string sender_;
    std::string target_;
    std::string body_;
    char msgType_ = '0';
    int seqNum_ = 1;
};

// ============================================================================
// Helper: Format message for display (replace SOH with |)
// ============================================================================

std::string formatForDisplay(const std::string& msg) {
    std::string result = msg;
    for (char& c : result) {
        if (c == SOH) c = '|';
    }
    return result;
}

// ============================================================================
// Example Messages
// ============================================================================

std::string createLogon(const std::string& sender,
                        const std::string& target,
                        int seqNum,
                        int heartbeatInterval = 30) {
    return FIXMessageWriter()
        .setMsgType(MsgType::Logon)
        .setSender(sender)
        .setTarget(target)
        .setSeqNum(seqNum)
        .setField(Tag::EncryptMethod, 0)
        .setField(Tag::HeartBtInt, heartbeatInterval)
        .build();
}

std::string createNewOrderSingle(const std::string& sender,
                                  const std::string& target,
                                  int seqNum,
                                  const std::string& clOrdID,
                                  const std::string& symbol,
                                  char side,
                                  int quantity,
                                  char ordType,
                                  double price = 0.0) {
    auto writer = FIXMessageWriter()
        .setMsgType(MsgType::NewOrderSingle)
        .setSender(sender)
        .setTarget(target)
        .setSeqNum(seqNum)
        .setField(Tag::ClOrdID, clOrdID)
        .setField(Tag::HandlInst, '1')  // Automated execution
        .setField(Tag::Symbol, symbol)
        .setField(Tag::Side, side)
        .setField(Tag::TransactTime, "20250120-10:30:00.000")
        .setField(Tag::OrderQty, quantity)
        .setField(Tag::OrdType, ordType);

    if (ordType == OrdType::Limit && price > 0) {
        writer.setField(Tag::Price, price);
    }

    return writer.build();
}

std::string createExecutionReport(const std::string& sender,
                                   const std::string& target,
                                   int seqNum,
                                   const std::string& orderID,
                                   const std::string& execID,
                                   const std::string& clOrdID,
                                   const std::string& symbol,
                                   char side,
                                   char ordStatus,
                                   char execType,
                                   int orderQty,
                                   int cumQty,
                                   int leavesQty,
                                   double avgPx,
                                   double lastPx = 0.0,
                                   int lastQty = 0) {
    auto writer = FIXMessageWriter()
        .setMsgType(MsgType::ExecutionReport)
        .setSender(sender)
        .setTarget(target)
        .setSeqNum(seqNum)
        .setField(Tag::OrderID, orderID)
        .setField(Tag::ExecID, execID)
        .setField(Tag::ClOrdID, clOrdID)
        .setField(Tag::ExecType, execType)
        .setField(Tag::OrdStatus, ordStatus)
        .setField(Tag::Symbol, symbol)
        .setField(Tag::Side, side)
        .setField(Tag::OrderQty, orderQty)
        .setField(Tag::CumQty, cumQty)
        .setField(Tag::LeavesQty, leavesQty)
        .setField(6, avgPx, 4);  // AvgPx (tag 6) with 4 decimal places

    if (lastQty > 0) {
        writer.setField(Tag::LastQty, lastQty);
        writer.setField(Tag::LastPx, lastPx, 4);
    }

    return writer.build();
}

std::string createOrderCancelRequest(const std::string& sender,
                                      const std::string& target,
                                      int seqNum,
                                      const std::string& origClOrdID,
                                      const std::string& clOrdID,
                                      const std::string& symbol,
                                      char side) {
    return FIXMessageWriter()
        .setMsgType(MsgType::OrderCancelReq)
        .setSender(sender)
        .setTarget(target)
        .setSeqNum(seqNum)
        .setField(Tag::OrigClOrdID, origClOrdID)
        .setField(Tag::ClOrdID, clOrdID)
        .setField(Tag::Symbol, symbol)
        .setField(Tag::Side, side)
        .setField(Tag::TransactTime, "20250120-10:31:00.000")
        .build();
}

// ============================================================================
// Demo: Complete Order Flow
// ============================================================================

void demonstrateOrderFlow() {
    std::cout << "=== FIX Protocol Order Flow Demo ===\n\n";

    // 1. Client sends Logon
    std::cout << "1. Client -> Broker: Logon\n";
    std::string logon = createLogon("CLIENT1", "BROKER1", 1);
    std::cout << "   " << formatForDisplay(logon) << "\n\n";

    // 2. Parse and display logon
    FIXMessageReader reader;
    if (reader.parse(logon)) {
        std::cout << "   Parsed Logon:\n";
        std::cout << "   - MsgType: " << reader.getChar(Tag::MsgType) << " (Logon)\n";
        std::cout << "   - Sender: " << reader.get(Tag::SenderCompID) << "\n";
        std::cout << "   - Target: " << reader.get(Tag::TargetCompID) << "\n";
        std::cout << "   - HeartBtInt: " << reader.getInt(Tag::HeartBtInt) << " seconds\n\n";
    }

    // 3. Client sends New Order Single
    std::cout << "2. Client -> Broker: New Order Single (Buy 1000 AAPL @ $150.25)\n";
    std::string newOrder = createNewOrderSingle(
        "CLIENT1", "BROKER1", 2,
        "ORD-001",    // ClOrdID
        "AAPL",       // Symbol
        Side::Buy,    // Side
        1000,         // Quantity
        OrdType::Limit,
        150.25        // Price
    );
    std::cout << "   " << formatForDisplay(newOrder) << "\n\n";

    // Parse new order
    if (reader.parse(newOrder)) {
        std::cout << "   Parsed Order:\n";
        std::cout << "   - ClOrdID: " << reader.get(Tag::ClOrdID) << "\n";
        std::cout << "   - Symbol: " << reader.get(Tag::Symbol) << "\n";
        std::cout << "   - Side: " << (reader.getChar(Tag::Side) == '1' ? "Buy" : "Sell") << "\n";
        std::cout << "   - Quantity: " << reader.getInt(Tag::OrderQty) << "\n";
        std::cout << "   - Price: $" << reader.getDouble(Tag::Price) << "\n\n";
    }

    // 4. Broker sends Execution Report (New)
    std::cout << "3. Broker -> Client: Execution Report (Order Acknowledged)\n";
    std::string execNew = createExecutionReport(
        "BROKER1", "CLIENT1", 2,
        "EXCH-12345",  // OrderID
        "EXEC-001",    // ExecID
        "ORD-001",     // ClOrdID
        "AAPL",        // Symbol
        Side::Buy,
        OrdStatus::New,
        ExecType::New,
        1000, 0, 1000, // orderQty, cumQty, leavesQty
        0.0            // avgPx
    );
    std::cout << "   " << formatForDisplay(execNew) << "\n\n";

    // 5. Broker sends Execution Report (Partial Fill)
    std::cout << "4. Broker -> Client: Execution Report (Partial Fill: 500 @ $150.20)\n";
    std::string execPartial = createExecutionReport(
        "BROKER1", "CLIENT1", 3,
        "EXCH-12345",
        "EXEC-002",
        "ORD-001",
        "AAPL",
        Side::Buy,
        OrdStatus::PartialFill,
        ExecType::Trade,
        1000, 500, 500,  // orderQty, cumQty, leavesQty
        150.20,          // avgPx
        150.20,          // lastPx
        500              // lastQty
    );
    std::cout << "   " << formatForDisplay(execPartial) << "\n\n";

    if (reader.parse(execPartial)) {
        std::cout << "   Parsed Execution:\n";
        std::cout << "   - OrderID: " << reader.get(Tag::OrderID) << "\n";
        std::cout << "   - Status: PartialFill\n";
        std::cout << "   - Filled: " << reader.getInt(Tag::CumQty) << "\n";
        std::cout << "   - Remaining: " << reader.getInt(Tag::LeavesQty) << "\n";
        std::cout << "   - Last Fill: " << reader.getInt(Tag::LastQty)
                  << " @ $" << reader.getDouble(Tag::LastPx) << "\n\n";
    }

    // 6. Broker sends Execution Report (Full Fill)
    std::cout << "5. Broker -> Client: Execution Report (Filled: 500 @ $150.25)\n";
    std::string execFilled = createExecutionReport(
        "BROKER1", "CLIENT1", 4,
        "EXCH-12345",
        "EXEC-003",
        "ORD-001",
        "AAPL",
        Side::Buy,
        OrdStatus::Filled,
        ExecType::Trade,
        1000, 1000, 0,   // orderQty, cumQty, leavesQty
        150.225,         // avgPx
        150.25,          // lastPx
        500              // lastQty
    );
    std::cout << "   " << formatForDisplay(execFilled) << "\n\n";

    if (reader.parse(execFilled)) {
        std::cout << "   Order Complete!\n";
        std::cout << "   - Total Filled: " << reader.getInt(Tag::CumQty) << " shares\n";
        std::cout << "   - Average Price: $" << reader.getDouble(6) << "\n\n";
    }

    // 7. Demo Cancel Request
    std::cout << "6. Example: Order Cancel Request\n";
    std::string cancelReq = createOrderCancelRequest(
        "CLIENT1", "BROKER1", 5,
        "ORD-001",      // Original order
        "CANCEL-001",   // Cancel request ID
        "AAPL",
        Side::Buy
    );
    std::cout << "   " << formatForDisplay(cancelReq) << "\n";
}

// ============================================================================
// Demo: Parse Sample Messages
// ============================================================================

void demonstrateParsing() {
    std::cout << "\n=== FIX Message Parsing Demo ===\n\n";

    // Build a sample message with correct checksum
    std::string sampleMsg = createExecutionReport(
        "BROKER", "CLIENT", 42,
        "ORDER123",
        "EXEC456",
        "MYORDER789",
        "MSFT",
        Side::Buy,
        OrdStatus::Filled,
        ExecType::Trade,
        5000, 5000, 0,
        425.50,
        425.50,
        2500
    );

    std::cout << "Raw message:\n" << formatForDisplay(sampleMsg) << "\n\n";

    FIXMessageReader reader;
    if (reader.parse(sampleMsg)) {
        std::cout << "Successfully parsed!\n\n";

        std::cout << "Header:\n";
        std::cout << "  BeginString: " << reader.get(Tag::BeginString) << "\n";
        std::cout << "  BodyLength: " << reader.getInt(Tag::BodyLength) << "\n";
        std::cout << "  MsgType: " << reader.getChar(Tag::MsgType) << " (Execution Report)\n";
        std::cout << "  MsgSeqNum: " << reader.getInt(Tag::MsgSeqNum) << "\n";
        std::cout << "  Sender: " << reader.get(Tag::SenderCompID) << "\n";
        std::cout << "  Target: " << reader.get(Tag::TargetCompID) << "\n";

        std::cout << "\nExecution Details:\n";
        std::cout << "  OrderID: " << reader.get(Tag::OrderID) << "\n";
        std::cout << "  ExecID: " << reader.get(Tag::ExecID) << "\n";
        std::cout << "  ClOrdID: " << reader.get(Tag::ClOrdID) << "\n";
        std::cout << "  Symbol: " << reader.get(Tag::Symbol) << "\n";
        std::cout << "  Side: " << (reader.getChar(Tag::Side) == '1' ? "Buy" : "Sell") << "\n";
        std::cout << "  OrderQty: " << reader.getInt(Tag::OrderQty) << "\n";
        std::cout << "  CumQty: " << reader.getInt(Tag::CumQty) << "\n";
        std::cout << "  LeavesQty: " << reader.getInt(Tag::LeavesQty) << "\n";
        std::cout << "  AvgPx: $" << std::fixed << std::setprecision(4)
                  << reader.getDouble(6) << "\n";
        std::cout << "  LastQty: " << reader.getInt(Tag::LastQty) << "\n";
        std::cout << "  LastPx: $" << reader.getDouble(Tag::LastPx) << "\n";

        char ordStatus = reader.getChar(Tag::OrdStatus);
        std::cout << "\n  Status: ";
        switch (ordStatus) {
            case '0': std::cout << "New"; break;
            case '1': std::cout << "Partial Fill"; break;
            case '2': std::cout << "Filled"; break;
            case '4': std::cout << "Canceled"; break;
            case '8': std::cout << "Rejected"; break;
            default: std::cout << "Unknown (" << ordStatus << ")"; break;
        }
        std::cout << "\n";

        std::cout << "\nTrailer:\n";
        std::cout << "  CheckSum: " << reader.get(Tag::CheckSum) << "\n";
    } else {
        std::cout << "Failed to parse message!\n";
    }
}

// ============================================================================
// Main
// ============================================================================

int main() {
    demonstrateOrderFlow();
    demonstrateParsing();

    std::cout << "\n=== Summary ===\n";
    std::cout << "This example demonstrated:\n";
    std::cout << "  1. FIX message structure (Header + Body + Trailer)\n";
    std::cout << "  2. Common message types (Logon, NewOrderSingle, ExecutionReport)\n";
    std::cout << "  3. Tag=Value parsing with SOH delimiter\n";
    std::cout << "  4. Checksum calculation and validation\n";
    std::cout << "  5. A complete order flow: submit -> ack -> partial fill -> fill\n";

    return 0;
}
