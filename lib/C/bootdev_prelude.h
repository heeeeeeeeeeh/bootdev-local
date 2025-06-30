// ~/Projects/Languages/C/LearningFr/bootdev_prelude.h
// This header provides compatibility macros for Boot.dev's custom munit syntax.
// It is designed to be force-included by GCC using the -include flag.

#ifndef BOOTDEV_PRELUDE_H
#define BOOTDEV_PRELUDE_H

// 1. Include the standard munit.h first. This provides the Munit structs,
//    and the underlying _full functions like munit_assert_type_full, munit_assert_int_full, etc.
#include "munit.h"

// 2. IMPORTANT: Undefine the standard munit assertion macros that Boot.dev expects to be 4-argument.
//    This clears the way for our 4-argument redefinitions.
#undef munit_assert_size
#undef munit_assert_uint8
#undef munit_assert_uint16
#undef munit_assert_uint32
#undef munit_assert_uint64
#undef munit_assert_int
#undef munit_assert_string_equal


// 3. Now, redefine these macros to match Boot.dev's 4-argument signature,
//    and make them call the appropriate _full functions from standard munit.
//    __FILE__ and __LINE__ are standard C preprocessor macros for filename and line number.

#define munit_assert_size(A, OP, B, MSG)    munit_assert_size_full(A, OP, B, __FILE__, __LINE__, MSG)
#define munit_assert_uint8(A, OP, B, MSG)   munit_assert_uint8_full(A, OP, B, __FILE__, __LINE__, MSG)
#define munit_assert_uint16(A, OP, B, MSG)  munit_assert_uint16_full(A, OP, B, __FILE__, __LINE__, MSG)
#define munit_assert_uint32(A, OP, B, MSG)  munit_assert_uint32_full(A, OP, B, __FILE__, __LINE__, MSG)
#define munit_assert_uint64(A, OP, B, MSG)  munit_assert_uint64_full(A, OP, B, __FILE__, __LINE__, MSG)
#define munit_assert_int(A, OP, B, MSG)     munit_assert_int_full(A, OP, B, __FILE__, __LINE__, MSG)
#define munit_assert_string_equal(A, B, MSG) munit_assert_string_equal_full(A, B, __FILE__, __LINE__, MSG)


// --- Compatibility for test functions and test suite definition (these should be fine) ---

// Emulates Boot.dev's 'munit_case(TYPE, NAME, { ... })'
#define munit_case(TYPE, NAME, CODE_BLOCK) \
    static MunitResult NAME(const MunitParameter params[], void* user_data) { \
        (void)params;   /* Suppress unused parameter warning */ \
        (void)user_data; /* Suppress unused parameter warning */ \
        CODE_BLOCK;     /* Execute the test code block */ \
        return MUNIT_OK; /* Return MUNIT_OK if assertions pass */ \
    }

// Emulates Boot.dev's 'assert_int(A, OP, B, MSG)'
// This will now correctly call our 4-arg munit_assert_int macro defined above.
#define assert_int(A, OP, B, MSG) munit_assert_int(A, OP, B, MSG)

// Emulates Boot.dev's 'assert_string_equal(ACTUAL, EXPECTED, MSG)'
// This will now correctly call our 4-arg munit_assert_string_equal macro defined above.
#define assert_string_equal(ACTUAL, EXPECTED, MSG) munit_assert_string_equal(ACTUAL, EXPECTED, MSG)


// Emulates Boot.dev's 'munit_test("/path", test_func)'
#define munit_test(PATH, TEST_FUNC_PTR) \
    { PATH, TEST_FUNC_PTR, NULL, NULL, 0, MUNIT_TEST_OPTION_NONE } /* Using 0 for teardown based on your specific munit.h */

// Emulates Boot.dev's 'munit_null_test'
#define munit_null_test \
    { NULL, NULL, NULL, NULL, 0, MUNIT_TEST_OPTION_NONE }

// Emulates Boot.dev's 'MunitSuite suite = munit_suite("name", tests);'
#define munit_suite(NAME_STR, TESTS_ARRAY_PTR) \
    (const MunitSuite) { \
        "/" NAME_STR,     /* Suite path/name */ \
        TESTS_ARRAY_PTR,  /* Pointer to the array of MunitTest */ \
        NULL,             /* Sub-suites (NULL if none) */ \
        1,                /* Number of setups/teardowns per test (usually 1) */ \
        MUNIT_SUITE_OPTION_NONE \
    }

#endif // BOOTDEV_PRELUDE_H