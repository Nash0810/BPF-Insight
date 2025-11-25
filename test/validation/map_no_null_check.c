// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, __u64);
} test_map SEC(".maps");

SEC("xdp")
int map_no_null_check(struct xdp_md *ctx)
{
    __u32 key = 0;

    __u64 *val = bpf_map_lookup_elem(&test_map, &key); // helper: lookup

    // ❌ no null check
    return *val;
}

char _license[] SEC("license") = "GPL";
