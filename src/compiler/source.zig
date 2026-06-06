const std = @import("std");

pub const SourceError = error{
    OffsetOutOfRange,
    SpanOutOfRange,
};

/// A 1-based diagnostic position in source text.
///
/// Columns are byte columns for now, not Unicode scalar, width, or grapheme
/// columns. The lexer/parser can refine this later if diagnostics need richer
/// Unicode-aware presentation.
pub const SourceLocation = struct {
    line: usize,
    column: usize,
    offset: usize,
};

/// A half-open byte range in a source file: [start, start + length).
pub const SourceSpan = struct {
    start: usize,
    length: usize,

    pub fn end(self: SourceSpan) ?usize {
        return std.math.add(usize, self.start, self.length) catch null;
    }
};

/// Borrowed source text plus precomputed line start offsets.
///
/// `display_name` and `text` are borrowed from the caller. `line_starts` is
/// owned by this value and must be released with `deinit`.
pub const SourceFile = struct {
    display_name: []const u8,
    text: []const u8,
    line_starts: []const usize,

    pub fn init(allocator: std.mem.Allocator, display_name: []const u8, text: []const u8) !SourceFile {
        return .{
            .display_name = display_name,
            .text = text,
            .line_starts = try computeLineStarts(allocator, text),
        };
    }

    pub fn deinit(self: SourceFile, allocator: std.mem.Allocator) void {
        allocator.free(self.line_starts);
    }

    pub fn len(self: SourceFile) usize {
        return self.text.len;
    }

    pub fn slice(self: SourceFile, span: SourceSpan) SourceError![]const u8 {
        const span_end = span.end() orelse return SourceError.SpanOutOfRange;
        if (span.start > self.text.len or span_end > self.text.len) {
            return SourceError.SpanOutOfRange;
        }

        return self.text[span.start..span_end];
    }

    pub fn locationAt(self: SourceFile, offset: usize) SourceError!SourceLocation {
        if (offset > self.text.len) {
            return SourceError.OffsetOutOfRange;
        }

        const line_index = self.lineIndexForOffset(offset);
        const line_start = self.line_starts[line_index];
        return .{
            .line = line_index + 1,
            .column = offset - line_start + 1,
            .offset = offset,
        };
    }

    pub fn spanStartLocation(self: SourceFile, span: SourceSpan) SourceError!SourceLocation {
        return self.locationAt(span.start);
    }

    fn lineIndexForOffset(self: SourceFile, offset: usize) usize {
        var low: usize = 0;
        var high: usize = self.line_starts.len;

        while (low < high) {
            const mid = low + (high - low) / 2;
            if (self.line_starts[mid] <= offset) {
                low = mid + 1;
            } else {
                high = mid;
            }
        }

        return low - 1;
    }
};

fn computeLineStarts(allocator: std.mem.Allocator, text: []const u8) ![]usize {
    var starts = std.ArrayList(usize).init(allocator);
    errdefer starts.deinit();

    try starts.append(0);

    var index: usize = 0;
    while (index < text.len) : (index += 1) {
        if (text[index] == '\n') {
            try starts.append(index + 1);
        }
    }

    return starts.toOwnedSlice();
}

test "empty source maps EOF to first line and column" {
    const source = try SourceFile.init(std.testing.allocator, "empty.con", "");
    defer source.deinit(std.testing.allocator);

    try std.testing.expectEqualStrings("empty.con", source.display_name);
    try std.testing.expectEqual(@as(usize, 0), source.len());
    try std.testing.expectEqual(@as(usize, 1), source.line_starts.len);
    try std.testing.expectEqual(@as(usize, 0), source.line_starts[0]);

    const location = try source.locationAt(0);
    try std.testing.expectEqual(@as(usize, 1), location.line);
    try std.testing.expectEqual(@as(usize, 1), location.column);
    try std.testing.expectEqual(@as(usize, 0), location.offset);
}

test "single-line source uses one-based byte columns" {
    const source = try SourceFile.init(std.testing.allocator, "single.con", "hello");
    defer source.deinit(std.testing.allocator);

    const start = try source.locationAt(0);
    try std.testing.expectEqual(@as(usize, 1), start.line);
    try std.testing.expectEqual(@as(usize, 1), start.column);

    const middle = try source.locationAt(4);
    try std.testing.expectEqual(@as(usize, 1), middle.line);
    try std.testing.expectEqual(@as(usize, 5), middle.column);
}

test "multi-line LF source maps line starts" {
    const source = try SourceFile.init(std.testing.allocator, "lf.con", "one\ntwo\nthree");
    defer source.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 3), source.line_starts.len);
    try std.testing.expectEqual(@as(usize, 0), source.line_starts[0]);
    try std.testing.expectEqual(@as(usize, 4), source.line_starts[1]);
    try std.testing.expectEqual(@as(usize, 8), source.line_starts[2]);

    const location = try source.locationAt(8);
    try std.testing.expectEqual(@as(usize, 3), location.line);
    try std.testing.expectEqual(@as(usize, 1), location.column);
}

test "multi-line CRLF source treats CRLF as one newline" {
    const source = try SourceFile.init(std.testing.allocator, "crlf.con", "one\r\ntwo\r\nthree");
    defer source.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 3), source.line_starts.len);
    try std.testing.expectEqual(@as(usize, 0), source.line_starts[0]);
    try std.testing.expectEqual(@as(usize, 5), source.line_starts[1]);
    try std.testing.expectEqual(@as(usize, 10), source.line_starts[2]);

    const location = try source.locationAt(10);
    try std.testing.expectEqual(@as(usize, 3), location.line);
    try std.testing.expectEqual(@as(usize, 1), location.column);
}

test "EOF offset maps to final line and column" {
    const source = try SourceFile.init(std.testing.allocator, "eof.con", "abc\ndef");
    defer source.deinit(std.testing.allocator);

    const location = try source.locationAt(source.len());
    try std.testing.expectEqual(@as(usize, 2), location.line);
    try std.testing.expectEqual(@as(usize, 4), location.column);
    try std.testing.expectEqual(source.len(), location.offset);
}

test "EOF after trailing newline maps to empty final line" {
    const source = try SourceFile.init(std.testing.allocator, "trailing.con", "abc\n");
    defer source.deinit(std.testing.allocator);

    const location = try source.locationAt(source.len());
    try std.testing.expectEqual(@as(usize, 2), location.line);
    try std.testing.expectEqual(@as(usize, 1), location.column);
}

test "span text extraction returns exact byte slice" {
    const source = try SourceFile.init(std.testing.allocator, "span.con", "prefix target suffix");
    defer source.deinit(std.testing.allocator);

    const span = SourceSpan{ .start = 7, .length = 6 };
    try std.testing.expectEqualStrings("target", try source.slice(span));
}

test "span across lines extracts newline bytes" {
    const source = try SourceFile.init(std.testing.allocator, "across.con", "first\nsecond\nthird");
    defer source.deinit(std.testing.allocator);

    const span = SourceSpan{ .start = 3, .length = 10 };
    try std.testing.expectEqualStrings("st\nsecond\n", try source.slice(span));

    const location = try source.spanStartLocation(span);
    try std.testing.expectEqual(@as(usize, 1), location.line);
    try std.testing.expectEqual(@as(usize, 4), location.column);
}

test "out-of-range offsets and spans fail" {
    const source = try SourceFile.init(std.testing.allocator, "invalid.con", "text");
    defer source.deinit(std.testing.allocator);

    try std.testing.expectError(SourceError.OffsetOutOfRange, source.locationAt(5));
    try std.testing.expectError(SourceError.SpanOutOfRange, source.slice(.{ .start = 3, .length = 2 }));
    try std.testing.expectError(SourceError.SpanOutOfRange, source.slice(.{ .start = std.math.maxInt(usize), .length = 1 }));
}
