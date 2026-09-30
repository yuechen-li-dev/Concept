// HLSL twin of scale.comp: destination[i] = source[i] * scale for i < count.
// Compiled with DXC (the SDSL-V back end); `concept vulkan-bind` reflects the
// same interface from it as from the GLSL build.
struct Constants
{
    uint count;
    float scale;
};

[[vk::push_constant]] Constants constants;
[[vk::binding(0, 0)]] StructuredBuffer<float> source;
[[vk::binding(1, 0)]] RWStructuredBuffer<float> destination;

[numthreads(64, 1, 1)]
void main(uint3 id : SV_DispatchThreadID)
{
    if (id.x < constants.count)
    {
        destination[id.x] = source[id.x] * constants.scale;
    }
}
