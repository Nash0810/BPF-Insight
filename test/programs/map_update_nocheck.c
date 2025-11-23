#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 4);
    __type(key, int);
    __type(value, int);
} test_map SEC(".maps");

SEC("xdp")
int prog(struct xdp_md *ctx)
{
    int key = bpf_get_prandom_u32();   // ❌ untrusted
    int value = 1;

    bpf_map_update_elem(&test_map, &key, &value, 0); // ❌ key unverifiable

    return 0;
}

char _license[] SEC("license") = "GPL";
