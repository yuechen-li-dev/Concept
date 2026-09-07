const std = @import("std");
const diagnostics = @import("diagnostics.zig");

/// P9-M7 capability annotation scaffold for compile-time functions.
///
/// Source spelling remains snake_case inside `comptime(...)` lists. These
/// capabilities only record requested host-visible effects. P9-M7 grants none
/// of them and exposes no host APIs; a future build manifest/configuration model
/// must make grants visible and auditable before any effect can execute.
pub const CompileTimeCapability = enum {
    read_fs,
    write_fs,
    env,
    network,
    time,
    random,
    process,

    pub fn fromName(source_name: []const u8) ?CompileTimeCapability {
        inline for (std.meta.fields(CompileTimeCapability)) |field| {
            if (std.mem.eql(u8, source_name, field.name)) return @enumFromInt(field.value);
        }
        return null;
    }

    pub fn name(self: CompileTimeCapability) []const u8 {
        return @tagName(self);
    }
};

pub const CompileTimeCapabilityRequired = struct {
    name: []const u8,
    span: diagnostics.SourceSpan,

    pub fn resolved(self: CompileTimeCapabilityRequired) ?CompileTimeCapability {
        return CompileTimeCapability.fromName(self.name);
    }
};

pub const CompileTimeCapabilitySet = struct {
    values: std.EnumSet(CompileTimeCapability) = .initEmpty(),

    pub fn contains(self: CompileTimeCapabilitySet, capability: CompileTimeCapability) bool {
        return self.values.contains(capability);
    }

    pub fn insert(self: *CompileTimeCapabilitySet, capability: CompileTimeCapability) bool {
        const had_capability = self.values.contains(capability);
        self.values.insert(capability);
        return !had_capability;
    }
};

pub const CompileTimeCapabilityList = []const CompileTimeCapabilityRequired;

pub fn hasDuplicateCapability(capabilities: CompileTimeCapabilityList) ?CompileTimeCapabilityRequired {
    for (capabilities, 0..) |capability, index| {
        for (capabilities[0..index]) |previous| {
            if (std.mem.eql(u8, capability.name, previous.name)) return capability;
        }
    }
    return null;
}

test "compile-time capability names resolve from source spelling" {
    try std.testing.expectEqual(CompileTimeCapability.read_fs, CompileTimeCapability.fromName("read_fs").?);
    try std.testing.expectEqual(CompileTimeCapability.process, CompileTimeCapability.fromName("process").?);
    try std.testing.expect(CompileTimeCapability.fromName("database") == null);
    try std.testing.expectEqualStrings("write_fs", CompileTimeCapability.write_fs.name());
}

test "compile-time capability duplicate detection uses source names" {
    const span: diagnostics.SourceSpan = .{ .start = 0, .length = 1 };
    const list = [_]CompileTimeCapabilityRequired{
        .{ .name = "read_fs", .span = span },
        .{ .name = "env", .span = span },
        .{ .name = "read_fs", .span = span },
    };
    try std.testing.expectEqualStrings("read_fs", hasDuplicateCapability(&list).?.name);
}
