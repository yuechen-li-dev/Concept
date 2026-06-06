const std = @import("std");

pub const ast = @import("ast.zig");
pub const diagnostics = @import("diagnostics.zig");
pub const interner = @import("interner.zig");
pub const hir = @import("hir.zig");
pub const types = @import("types.zig");
pub const lexer = @import("lexer.zig");
pub const parser = @import("parser.zig");
pub const source = @import("source.zig");
pub const token = @import("token.zig");
pub const testing = @import("testing/harness.zig");
pub const checker = @import("checker.zig");
pub const backend_c = @import("backend_c.zig");

test {
    _ = std;
    _ = interner;
    _ = hir;
    _ = types;
}
