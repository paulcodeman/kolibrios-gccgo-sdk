#ifndef KOLIBRI_RUNTIME_ENTROPY_H
#define KOLIBRI_RUNTIME_ENTROPY_H
/* GCC's upstream CPUID implementation and OpenSSL's unchanged byte routines.
 * Only CPU feature selection is specific to the freestanding OS backend.
 * There is no clock/PID fallback for cryptographic callers. */
#include <cpuid.h>

extern unsigned int OPENSSL_ia32_rdseed_bytes(unsigned char *, unsigned int);
extern unsigned int OPENSSL_ia32_rdrand_bytes(unsigned char *, unsigned int);

int32_t runtime_kolibri_read_random(unsigned char *, int32_t)
    __asm__("runtime.kolibriReadRandom");
int32_t runtime_kolibri_read_random(unsigned char *output, int32_t length)
{
    if (length<=0) return 0;
    unsigned int a,b,c,d;
    int rdrand=__get_cpuid(1,&a,&b,&c,&d) && (c&bit_RDRND);
    int rdseed=__get_cpuid_count(7,0,&a,&b,&c,&d) && (b&bit_RDSEED);
    unsigned int filled=0;
    if (rdseed) filled=OPENSSL_ia32_rdseed_bytes(output,(unsigned int)length);
    if (rdrand && filled<(unsigned int)length)
        filled+=OPENSSL_ia32_rdrand_bytes(output+filled,(unsigned int)length-filled);
    return (int32_t)filled;
}
#endif
