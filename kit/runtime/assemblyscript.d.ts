/**
 * AssemblyScript built-in types for IDE support.
 *
 * AS uses i32/u32/f32/f64/bool as primitive types. These don't exist in
 * standard TypeScript. This file declares them as type aliases so the IDE
 * doesn't complain about AS fixture files.
 *
 * These are NOT used at runtime — the AS compiler handles them natively.
 */

declare type i8 = number;
declare type i16 = number;
declare type i32 = number;
declare type i64 = number;
declare type u8 = number;
declare type u16 = number;
declare type u32 = number;
declare type u64 = number;
declare type f32 = number;
declare type f64 = number;
declare type bool = boolean;
declare type usize = number;
declare type isize = number;

/** AssemblyScript I32 static methods. */
declare namespace I32 {
  function parseInt(s: string, radix?: number): i32;
}

/** AssemblyScript I64 static methods. */
declare namespace I64 {
  function parseInt(s: string, radix?: number): i64;
}

/** AssemblyScript F32 static methods. */
declare namespace F32 {
  function parseFloat(s: string): f32;
}

/** AssemblyScript F64 static methods. */
declare namespace F64 {
  function parseFloat(s: string): f64;
}

/** AssemblyScript unsafe type cast. */
declare function changetype<T>(value: any): T;

/** AssemblyScript sizeof operator. */
declare function sizeof<T>(): usize;

/** AssemblyScript assert. */
declare function assert<T>(condition: T, message?: string): T;
