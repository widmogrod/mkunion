// Driver for swift_roundtrip_test.go: not exporter output, the golden test
// ignores this directory. It reads {"items": [{"kind": ..., "value": ...}]}
// from stdin, decodes every value with the generated types for its kind
// (query, answer or status), encodes it again and writes the same shape to
// stdout. A value the union decoder refuses with MkunionJSON.Error comes back
// as {"kind": ..., "error": "noType" | "unknownType"}; any other failure is
// written to stderr and the driver exits 1.

import Foundation

enum Decoded {
    case query(Testasset.Query?)
    case answer(Testasset.Answer?)
    case status(Testasset.Status?)
    case refused(MkunionJSON.Error)
}

struct Item: Codable {
    var kind: String
    var decoded: Decoded

    enum CodingKeys: String, CodingKey {
        case kind, value, error
    }

    init(from decoder: any Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        kind = try c.decode(String.self, forKey: .kind)
        do {
            switch kind {
            case "query": decoded = .query(try c.decodeIfPresent(Testasset.Query.self, forKey: .value))
            case "answer": decoded = .answer(try c.decodeIfPresent(Testasset.Answer.self, forKey: .value))
            case "status": decoded = .status(try c.decodeIfPresent(Testasset.Status.self, forKey: .value))
            default:
                throw DecodingError.dataCorruptedError(forKey: .kind, in: c, debugDescription: "unknown kind \(kind)")
            }
        } catch let error as MkunionJSON.Error {
            decoded = .refused(error)
        }
    }

    func encode(to encoder: any Encoder) throws {
        var c = encoder.container(keyedBy: CodingKeys.self)
        try c.encode(kind, forKey: .kind)
        switch decoded {
        case .query(let value): try c.encode(value, forKey: .value)
        case .answer(let value): try c.encode(value, forKey: .value)
        case .status(let value): try c.encode(value, forKey: .value)
        case .refused(.noType): try c.encode("noType", forKey: .error)
        case .refused(.unknownType): try c.encode("unknownType", forKey: .error)
        }
    }
}

struct Batch: Codable {
    var items: [Item]
}

let input = FileHandle.standardInput.readDataToEndOfFile()
do {
    let batch = try MkunionJSON.decoder().decode(Batch.self, from: input)
    FileHandle.standardOutput.write(try MkunionJSON.encoder().encode(batch))
} catch {
    FileHandle.standardError.write(Data("driver: \(error)\n".utf8))
    exit(1)
}
