#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 4);
    __type(key, int);
    __type(value, int);
} test_map SEC(".maps");

SEC("xdp")
int prog(struct xdp_md *ctx)
{
    int key = 0;
    int *val = bpf_map_lookup_elem(&test_map, &key);

    // ❌ No null check
    return *val; 
}

char _license[] SEC("license") = "GPL";
