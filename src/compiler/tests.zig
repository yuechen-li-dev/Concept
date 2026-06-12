const std = @import("std");

pub const ast = @import("ast.zig");
pub const diagnostics = @import("diagnostics.zig");
pub const interner = @import("interner.zig");
pub const hir = @import("hir.zig");
pub const mir = @import("mir.zig");
pub const mir_lowering = @import("mir_lowering.zig");
pub const mir_validator = @import("mir_validator.zig");
pub const manifest = @import("manifest.zig");
pub const types = @import("types.zig");
pub const semantics = @import("semantics.zig");
pub const hir_checker = @import("hir_checker.zig");
pub const lexer = @import("lexer.zig");
pub const parser = @import("parser.zig");
pub const source = @import("source.zig");
pub const token = @import("token.zig");
pub const testing = @import("testing/harness.zig");
pub const checker = @import("checker.zig");
pub const compile_time = @import("compile_time.zig");
pub const compile_time_target = @import("compile_time_target.zig");
pub const backend_c = @import("backend_c.zig");
pub const backend_c_mir = @import("backend_c_mir.zig");

test {
    _ = std;
    _ = interner;
    _ = hir;
    _ = mir;
    _ = mir_lowering;
    _ = mir_validator;
    _ = manifest;
    _ = types;
    _ = semantics;
    _ = hir_checker;
    _ = compile_time;
    _ = compile_time_target;
}
