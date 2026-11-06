/*
 * Copyright (C) 2016 The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package libcore.java.util;

import static org.junit.Assert.assertArrayEquals;
import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertNotSame;
import static org.junit.Assert.assertSame;
import static org.junit.Assert.assertThrows;
import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;


import java.lang.reflect.Method;
import java.util.Arrays;
import java.util.List;
import java.util.Random;
import java.util.stream.Collectors;
import java.util.stream.IntStream;

import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public class ArraysTest {
    private static final int[] TEST_ARRAY_SIZES = { 0, 1, 2, 10, 100, 1000 };


    /**
     * java.util.Arrays#setAll(int[], java.util.function.IntUnaryOperator)
     */
    @Test
    public void setAll$I() {
        int[] list = new int[3];
        list[0] = 0;
        list[1] = 1;
        list[2] = 2;

        Arrays.setAll(list, x -> x + 1);
        assertEquals(1, list[0]);
        assertEquals(2, list[1]);
        assertEquals(3, list[2]);

        assertThrows(NullPointerException.class, () -> Arrays.setAll(list, null));
        assertThrows(NullPointerException.class, () -> Arrays.setAll((int[]) null, (x -> x + 1)));
    }

    /**
     * java.util.Arrays#parallelSetAll(int[], java.util.function.IntUnaryOperator)
     */
    @Test
    public void parallelSetAll$I() {
        int[] list = new int[3];
        list[0] = 0;
        list[1] = 1;
        list[2] = 2;

        Arrays.parallelSetAll(list, x -> x + 1);
        assertEquals(1, list[0]);
        assertEquals(2, list[1]);
        assertEquals(3, list[2]);

        assertThrows(NullPointerException.class, () -> Arrays.parallelSetAll(list, null));
        assertThrows(NullPointerException.class, () -> Arrays.parallelSetAll((int[]) null, (x -> x + 1)));
    }

    /**
     * java.util.Arrays#setAll(long[], java.util.function.IntToLongFunction)
     */
    @Test
    public void setAll$L() {
        long[] list = new long[3];
        list[0] = 0;
        list[1] = 1;
        list[2] = 2;

        Arrays.setAll(list, x -> x + 1);
        assertEquals(1, list[0]);
        assertEquals(2, list[1]);
        assertEquals(3, list[2]);

        assertThrows(NullPointerException.class, () -> Arrays.setAll(list, null));
        assertThrows(NullPointerException.class, () -> Arrays.setAll((long[]) null, (x -> x + 1)));
    }

    /**
     * java.util.Arrays#parallelSetAll(long[], java.util.function.IntToLongFunction)
     */
    @Test
    public void parallelSetAll$L() {
        long[] list = new long[3];
        list[0] = 0;
        list[1] = 1;
        list[2] = 2;

        Arrays.parallelSetAll(list, x -> x + 1);
        assertEquals(1, list[0]);
        assertEquals(2, list[1]);
        assertEquals(3, list[2]);

        assertThrows(NullPointerException.class, () -> Arrays.parallelSetAll(list, null));
        assertThrows(NullPointerException.class, () ->
                Arrays.parallelSetAll((long[]) null, (x -> x + 1)));
    }

    /**
     * java.util.Arrays#setAll(double[], java.util.function.IntToDoubleFunction)
     */
    @Test
    public void setAll$D() {
        double[] list = new double[3];
        list[0] = 0.0d;
        list[1] = 1.0d;
        list[2] = 2.0d;

        Arrays.setAll(list, x -> x + 0.5);
        assertEquals(0.5d, list[0], 0.0);
        assertEquals(1.5d, list[1], 0.0);
        assertEquals(2.5d, list[2], 0.0);

        assertThrows(NullPointerException.class, () -> Arrays.setAll(list, null));
        assertThrows(NullPointerException.class, () ->
                Arrays.setAll((double[]) null, x -> x + 0.5));
    }

    /**
     * java.util.Arrays#parallelSetAll(double[], java.util.function.IntToDoubleFunction)
     */
    @Test
    public void parallelSetAll$D() {
        double[] list = new double[3];
        list[0] = 0.0d;
        list[1] = 1.0d;
        list[2] = 2.0d;

        Arrays.parallelSetAll(list, x -> x + 0.5);
        assertEquals(0.5d, list[0], 0.0);
        assertEquals(1.5d, list[1], 0.0);
        assertEquals(2.5d, list[2], 0.0);

        assertThrows(NullPointerException.class, () -> Arrays.parallelSetAll(list, null));
        assertThrows(NullPointerException.class, () ->
                Arrays.parallelSetAll((double[]) null, x -> x + 0.5));
    }

    /**
     * java.util.Array#setAll(T[], java.util.function.IntFunction<\? extends T>)
     */
    @Test
    public void setAll$T() {
        String[] strings = new String[3];
        strings[0] = "a";
        strings[1] = "b";
        strings[2] = "c";

        Arrays.setAll(strings, x -> "a" + x);
        assertEquals("a0", strings[0]);
        assertEquals("a1", strings[1]);
        assertEquals("a2", strings[2]);

        assertThrows(NullPointerException.class, () -> Arrays.setAll(strings, null));
        assertThrows(NullPointerException.class, () ->
                Arrays.setAll((String[]) null, x -> "a" + x));
    }

    /**
     * java.util.Array#parallelSetAll(T[], java.util.function.IntFunction<\? extends T>)
     */
    @Test
    public void parallelSetAll$T() {
        String[] strings = new String[3];
        strings[0] = "a";
        strings[1] = "b";
        strings[2] = "c";

        Arrays.parallelSetAll(strings, x -> "a" + x);
        assertEquals("a0", strings[0]);
        assertEquals("a1", strings[1]);
        assertEquals("a2", strings[2]);

        assertThrows(NullPointerException.class, () -> Arrays.parallelSetAll(strings, null));
        assertThrows(NullPointerException.class, () ->
                Arrays.parallelSetAll((String[]) null, x -> "a" + x));
    }

    /**
     * java.util.Array#parallelPrefix(int[], java.util.function.IntBinaryOperator)
     */
    @Test
    public void parallelPrefix$I() {
        // Get an arbitrary array of ints.
        Random rand = new Random(0);
        int[] list = new int[1000];
        for(int i = 0; i < list.length; ++i) {
            list[i] = rand.nextInt() % 1000; // Prevent overflow
        }

        int[] seqResult = list.clone();

        // Sequential solution
        for(int i = 0; i < seqResult.length - 1; ++i) {
            seqResult[i + 1] += seqResult[i];
        }

        Arrays.parallelPrefix(list, (x, y) -> x + y);
        assertArrayEquals(seqResult, list);

        assertThrows(NullPointerException.class, () -> Arrays.parallelPrefix(list, null));
        assertThrows(NullPointerException.class, () ->
                Arrays.parallelPrefix((int[]) null, (x, y) -> x + y));
    }

    /**
     * java.util.Array#parallelPrefix(int[], int, int, java.util.function.IntBinaryOperator)
     */
    @Test
    public void parallelPrefix$III() {
        // Get an arbitrary array of ints.
        Random rand = new Random(0);
        int[] list = new int[1000];
        for(int i = 0; i < list.length; ++i) {
            list[i] = rand.nextInt() % 1000; // Prevent overflow
        }

        int begin = 100, end = 500;
        int[] seqResult = list.clone();

        // Sequential solution
        for(int i = begin; i < end - 1; ++i) {
            seqResult[i + 1] += seqResult[i];
        }

        Arrays.parallelPrefix(list, begin, end, (x, y) -> x + y);
        assertArrayEquals(seqResult, list);

        assertThrows(NullPointerException.class, () ->
                Arrays.parallelPrefix(list, begin, end, null));
        assertThrows(NullPointerException.class,
                () -> Arrays.parallelPrefix((int[]) null, begin, end, (x, y) -> x + y));
        assertThrows(IllegalArgumentException.class,
                () -> Arrays.parallelPrefix(list, end, begin, (x, y) -> x + y));
    }

    /**
     * java.util.Array#parallelPrefix(long[], java.util.function.LongBinaryOperator)
     */
    @Test
    public void parallelPrefix$L() {
        // Get an arbitrary array of ints.
        Random rand = new Random(0);
        long[] list = new long[1000];
        for(int i = 0; i < list.length; ++i) {
            list[i] = rand.nextLong() % 1000000; // Prevent overflow
        }

        long[] seqResult = list.clone();

        // Sequential solution
        for(int i = 0; i < seqResult.length - 1; ++i) {
            seqResult[i + 1] += seqResult[i];
        }

        Arrays.parallelPrefix(list, (x, y) -> x + y);
        assertArrayEquals(seqResult, list);

        assertThrows(NullPointerException.class, () -> Arrays.parallelPrefix(list, null));
        assertThrows(NullPointerException.class, () ->
                Arrays.parallelPrefix((long[]) null, (x, y) -> x + y));
    }

    /**
     * java.util.Array#parallelPrefix(long[], int, int, java.util.function.LongBinaryOperator)
     */
    @Test
    public void parallelPrefix$LII() {
        // Get an arbitrary array of ints.
        Random rand = new Random(0);
        long[] list = new long[1000];
        for(int i = 0; i < list.length; ++i) {
            list[i] = rand.nextLong() % 1000000; // Prevent overflow
        }

        int begin = 100, end = 500;
        long[] seqResult = list.clone();

        // Sequential solution
        for(int i = begin; i < end - 1; ++i) {
            seqResult[i + 1] += seqResult[i];
        }

        Arrays.parallelPrefix(list, begin, end, (x, y) -> x + y);
        assertArrayEquals(seqResult, list);

        assertThrows(NullPointerException.class, () ->
                Arrays.parallelPrefix(list, begin, end, null));
        assertThrows(NullPointerException.class,
                () -> Arrays.parallelPrefix((long[]) null, begin, end, (x, y) -> x + y));
        assertThrows(IllegalArgumentException.class,
                () -> Arrays.parallelPrefix(list, end, begin, (x, y) -> x + y));
    }

    /**
     * java.util.Array#parallelPrefix(double[], java.util.function.DoubleBinaryOperator)
     */
    @Test
    public void parallelPrefix$D() {
        // Get an arbitrary array of ints.
        Random rand = new Random(0);
        double[] list = new double[1000];
        for(int i = 0; i < list.length; ++i) {
            list[i] = rand.nextDouble() * 1000;
        }

        double[] seqResult = list.clone();

        // Sequential solution
        for(int i = 0; i < seqResult.length - 1; ++i) {
            seqResult[i + 1] += seqResult[i];
        }

        Arrays.parallelPrefix(list, (x, y) -> x + y);

        // Parallel double arithmetic contains error, reduce to integer for comparison.
        int[] listInInt = Arrays.stream(list).mapToInt(x -> (int) x).toArray();
        int[] seqResultInInt = Arrays.stream(seqResult).mapToInt(x -> (int) x).toArray();
        assertArrayEquals(seqResultInInt, listInInt);

        assertThrows(NullPointerException.class, () -> Arrays.parallelPrefix(list, null));
        assertThrows(NullPointerException.class, () ->
                Arrays.parallelPrefix((double[]) null, (x, y) -> x + y));
    }

    /**
     * java.util.Array#parallelPrefix(double[], int, int, java.util.function.DoubleBinaryOperator)
     */
    @Test
    public void parallelPrefix$DII() {
        // Get an arbitrary array of ints.
        Random rand = new Random(0);
        double[] list = new double[1000];
        for(int i = 0; i < list.length; ++i) {
            list[i] = rand.nextDouble() * 1000;
        }

        int begin = 100, end = 500;
        double[] seqResult = list.clone();

        // Sequential solution
        for(int i = begin; i < end - 1; ++i) {
            seqResult[i + 1] += seqResult[i];
        }

        Arrays.parallelPrefix(list, begin, end, (x, y) -> x + y);

        // Parallel double arithmetic contains error, reduce to integer for comparison.
        int[] listInInt = Arrays.stream(list).mapToInt(x -> (int) x).toArray();
        int[] seqResultInInt = Arrays.stream(seqResult).mapToInt(x -> (int) x).toArray();
        assertArrayEquals(seqResultInInt, listInInt);

        assertThrows(NullPointerException.class, () ->
                Arrays.parallelPrefix(list, begin, end, null));
        assertThrows(NullPointerException.class,
                () -> Arrays.parallelPrefix((double[]) null, begin, end, (x, y) -> x + y));
        assertThrows(IllegalArgumentException.class,
                () -> Arrays.parallelPrefix(list, end, begin, (x, y) -> x + y));
    }

    /**
     * java.util.Array#parallelPrefix(T[], java.util.function.BinaryOperator<T>)
     */
    @Test
    public void parallelPrefix$T() {
        String[] strings = new String[3];
        strings[0] = "a";
        strings[1] = "b";
        strings[2] = "c";

        Arrays.parallelPrefix(strings, (x, y) -> x + y);
        assertEquals("a", strings[0]);
        assertEquals("ab", strings[1]);
        assertEquals("abc", strings[2]);

        assertThrows(NullPointerException.class, () -> Arrays.parallelPrefix(strings, null));
        assertThrows(NullPointerException.class, () ->
                Arrays.parallelPrefix((String[]) null, (x, y) -> x + y));
    }

    /**
     * java.util.Array#parallelPrefix(T[], int, int, java.util.function.BinaryOperator<T>)
     */
    @Test
    public void parallelPrefix$TII() {
        String[] strings = new String[5];
        strings[0] = "a";
        strings[1] = "b";
        strings[2] = "c";
        strings[3] = "d";
        strings[4] = "e";
        int begin = 1, end = 4;

        Arrays.parallelPrefix(strings, begin, end, (x, y) -> x + y);
        assertEquals("a", strings[0]);
        assertEquals("b", strings[1]);
        assertEquals("bc", strings[2]);
        assertEquals("bcd", strings[3]);
        assertEquals("e", strings[4]);

        assertThrows(NullPointerException.class, () ->
                Arrays.parallelPrefix(strings, begin, end, null));
        assertThrows(NullPointerException.class,
                () -> Arrays.parallelPrefix((String[]) null, begin, end, (x, y) -> x + y));
        assertThrows(IllegalArgumentException.class,
                () -> Arrays.parallelPrefix(strings, end, begin, (x, y) -> x + y));
    }

    /**
     * java.util.Array#parallelPrefix(T[], int, int, java.util.function.BinaryOperator<T>)
     */
    @Test
    public void parallelPrefix$TII_biggerArray() {
        String[] strings = new String[1_000];
        int begin = 0, end = strings.length;

        for (int i = 0; i < strings.length; ++i) {
            strings[i] = String.valueOf(i);
        }

        Arrays.parallelPrefix(strings, begin, end, (x, y) -> x + y);

        String currentPrefix = "";
        for (int i = 0; i < strings.length; ++i) {
            currentPrefix += String.valueOf(i);

            assertEquals(currentPrefix, strings[i]);
        }
    }

    // http://b/74236526
    @Test
    public void deepEquals_nestedArraysOfDifferentTypesButEqualValues() {
        assertTrue(Arrays.deepEquals(
            new Object[] { new Object[] { "Hello", "world" } },
            new Object[] { new String[] { "Hello", "world" } }));
    }

    @Test
    public void streamInt() {
        for (int size : TEST_ARRAY_SIZES) {
            int[] sourceArray = intTestArray(size);

            // Stream, map, accumulate
            int sum = Arrays.stream(sourceArray)
                .map(i -> i + i)
                .sum();
            assertEquals(size * (size - 1), sum);

            // Stream, collect as array again
            int[] destArray = Arrays.stream(sourceArray)
                .toArray();
            assertArrayEquals(sourceArray, destArray);
            assertNotSame(sourceArray, destArray);

            // Stream, box, collect as list
            List<Integer> destList = Arrays.stream(sourceArray)
                .boxed()
                .collect(Collectors.toList());

            assertEquals(size, destList.size());
            for (int i = 0; i < size; i++) {
                assertEquals((int) destList.get(i), i);
            }
        }
    }

    @Test
    public void streamIntStartEnd() {
        final int size = 10;
        int[] sourceArray = intTestArray(size);
        for (int start = 0; start < size - 1; start++) {
            for (int end = start; end < size; end++) {
                int[] destArray = Arrays.stream(sourceArray, start, end)
                    .toArray();
                int len = end - start;
                assertEquals(len, destArray.length);
                if (len > 0) {
                    assertEquals(start, destArray[0]);
                    assertEquals(end - 1, destArray[len - 1]);
                }
            }
        }
    }

    @Test
    public void streamIntStartEnd_Exceptions() {
        int[] sourceArray = intTestArray(10);
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, -1, 9)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 0, 11)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 11, 11)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 0, -1)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 4, 3)
                .sum());
    }

    @Test
    public void streamLong() {
        for (int size : TEST_ARRAY_SIZES) {
            long[] sourceArray = longTestArray(size);

            // Stream, map, accumulate
            long sum = Arrays.stream(sourceArray)
                .map(i -> i + i)
                .sum();
            assertEquals(size * (size - 1), sum);

            // Stream, collect as array again
            long[] destArray = Arrays.stream(sourceArray)
                .toArray();
            assertArrayEquals(sourceArray, destArray);
            assertNotSame(sourceArray, destArray);

            // Stream, box, collect as list
            List<Long> destList = Arrays.stream(sourceArray)
                .boxed()
                .collect(Collectors.toList());

            assertEquals(size, destList.size());
            for (int i = 0; i < size; i++) {
                assertEquals((long) destList.get(i), i);
            }
        }
    }

    @Test
    public void streamLongStartEnd() {
        final int size = 10;
        long[] sourceArray = longTestArray(size);
        for (int start = 0; start < size - 1; start++) {
            for (int end = start; end < size; end++) {
                long[] destArray = Arrays.stream(sourceArray, start, end)
                    .toArray();
                int len = end - start;
                assertEquals(len, destArray.length);
                if (len > 0) {
                    assertEquals(start, destArray[0]);
                    assertEquals(end - 1, destArray[len - 1]);
                }
            }
        }
    }

    @Test
    public void streamLongStartEnd_Exceptions() {
        long[] sourceArray = longTestArray(10);
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, -1, 9)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 0, 11)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 11, 11)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 0, -1)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 4, 3)
                .sum());
    }

    @Test
    public void streamDouble() {
        for (int size : TEST_ARRAY_SIZES) {
            double[] sourceArray = doubleTestArray(size);

            // Stream, map, accumulate
            double sum = Arrays.stream(sourceArray)
                .map(i -> i + i)
                .sum();
            assertEquals(size * (size - 1), sum, 0.001);

            // Stream, collect as array again
            double[] destArray = Arrays.stream(sourceArray)
                .toArray();
            assertArrayEquals(sourceArray, destArray, 0.001);
            assertNotSame(sourceArray, destArray);

            // Stream, box, collect as list
            List<Double> destList = Arrays.stream(sourceArray)
                .boxed()
                .collect(Collectors.toList());

            assertEquals(size, destList.size());
            for (int i = 0; i < size; i++) {
                assertEquals(destList.get(i), i, 0.001);
            }
        }
    }

    @Test
    public void streamDoubleStartEnd() {
        final int size = 10;
        double[] sourceArray = doubleTestArray(size);
        for (int start = 0; start < size - 1; start++) {
            for (int end = start; end < size; end++) {
                double[] destArray = Arrays.stream(sourceArray, start, end)
                    .toArray();
                int len = end - start;
                assertEquals(len, destArray.length);
                if (len > 0) {
                    assertEquals(start, destArray[0], 0.0);
                    assertEquals(end - 1, destArray[len - 1], 0.0);
                }
            }
        }
    }

    @Test
    public void streamDoubleStartEnd_Exceptions() {
        double[] sourceArray = doubleTestArray(10);
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, -1, 9)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 0, 11)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 11, 11)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 0, -1)
                .sum());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 4, 3)
                .sum());
    }

    @Test
    public void streamObject() {
      for (int size : TEST_ARRAY_SIZES) {
        String[] sourceArray = stringTestArray(size);

        // Stream, map, accumulate
        int sum = Arrays.stream(sourceArray)
            .mapToInt(i -> Integer.parseInt(i) * 2)
            .sum();
        assertEquals(size * (size - 1), sum);

        // Stream, collect as array again
        String[] destArray = Arrays.stream(sourceArray)
            .toArray(String[]::new);
        assertArrayEquals(sourceArray, destArray);
        assertNotSame(sourceArray, destArray);

        // Stream, collect as list
        List<String> destList = Arrays.stream(sourceArray)
            .collect(Collectors.toList());

        assertEquals(size, destList.size());
        for (int i = 0; i < size; i++) {
          assertSame(destList.get(i), sourceArray[i]);
        }
      }
    }

    @Test
    public void streamObjectStartEnd() {
        final int size = 10;
        String[] sourceArray = stringTestArray(size);
        for (int start = 0; start < size - 1; start++) {
            for (int end = start; end < size; end++) {
                String[] destArray = Arrays.stream(sourceArray, start, end)
                    .toArray(String[]::new);
                int len = end - start;
                assertEquals(len, destArray.length);
                if (len > 0) {
                    assertSame(sourceArray[start], destArray[0]);
                    assertSame(sourceArray[end - 1], destArray[len - 1]);
                }
            }
        }
    }

    @Test
    public void streamObjectStartEnd_Exceptions() {
        String[] sourceArray = stringTestArray(10);
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, -1, 9)
                .count());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 0, 11)
                .count());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 11, 11)
                .count());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 0, -1)
                .count());
        assertThrows(ArrayIndexOutOfBoundsException.class, () -> Arrays.stream(sourceArray, 4, 3)
                .count());
    }

    @Test
    public void compareLIILII() {
        final Integer[] lhs = { Integer.valueOf(0), Integer.valueOf(1), Integer.valueOf(2) };
        final Integer[] rhs = { Integer.valueOf(1), Integer.valueOf(2), Integer.valueOf(0) };
        final Integer[] empty = (Integer []) null;

        assertEquals(0, Arrays.compare(lhs, 0, lhs.length, lhs, 0, lhs.length));
        assertEquals(0, Arrays.compare(lhs, 1, 3, rhs, 0, 2));
        assertEquals(-1, Arrays.compare(lhs, 0, 2, rhs, 0, 2));
        assertEquals(-1, Arrays.compare(lhs, 0, 2, rhs, 1, 3));
        assertEquals(-1, Arrays.compare(lhs, 0, 3, rhs, 0, 3));
        assertEquals(1, Arrays.compare(rhs, 0, 3, lhs, 0, 3));

        for (Integer[][] arrays : new Integer[][][] { { lhs, empty }, { empty, rhs }}) {
            assertThrows(NullPointerException.class,
                    () -> Arrays.compare(arrays[0], 1, 3, arrays[1], 0, 2));
        }

        for (int[] i : new int[][] {{3, 1, 0, 2}, {1, 3, 2, 0}}) {
            assertThrows(IllegalArgumentException.class,
                    () -> Arrays.compare(lhs, i[0], i[1], rhs, i[2], i[3]));
        }

        for (int[] i : new int[][] { {-1, 1, 0, 1},
                                     {0, lhs.length + 1, 0, 1},
                                     {0, 1, -1, 1},
                                     {0, 1, 0, rhs.length + 1}}) {
            assertThrows(ArrayIndexOutOfBoundsException.class,
                    () -> Arrays.compare(lhs, i[0], i[1], rhs, i[2], i[3]));
        }
    }

    @Test
    public void arraysArrayListToArray_componentType() throws Exception {
        List<String> strings = Arrays.asList("one", "two");

        Method toArrayWithComponentType =
                strings.getClass().getDeclaredMethod("toArrayPreserveComponentType");
        toArrayWithComponentType.setAccessible(true);

        assertEquals(String[].class, toArrayWithComponentType.invoke(strings).getClass());
    }

    @Test
    public void arraysArrayListToArray_noComponentType() throws Exception {
        List<String> strings = Arrays.asList("one", "two");

        Method toArrayWithoutComponentType =
                strings.getClass().getDeclaredMethod("toArrayWithoutComponentType");
        toArrayWithoutComponentType.setAccessible(true);

        assertEquals(Object[].class, toArrayWithoutComponentType.invoke(strings).getClass());
    }



    private int[] intTestArray(int size) {
        int[] array = new int[size];
        for (int i = 0; i < size; i++) {
            array[i] = i;
        }
        return array;
    }

    private long[] longTestArray(int size) {
        long[] array = new long[size];
        for (int i = 0; i < size; i++) {
            array[i] = i;
        }
        return array;
    }

    private double[] doubleTestArray(int size) {
        double[] array = new double[size];
        for (int i = 0; i < size; i++) {
            array[i] = i;
        }
        return array;
    }

    private String[] stringTestArray(int size) {
        String[] array = new String[size];
        for (int i = 0; i < size; i++) {
            array[i] = String.valueOf(i);
        }
        return array;
    }

    private byte[] byteTestArray(int size, int seed) {
        byte[] array = new byte[size];
        if (size > 0) {
            new Random(seed).nextBytes(array);
        }
        return array;
    }

    @Test
    public void testEqualsBooleanRange() {
        boolean[] a = {true, false, true};
        boolean[] b = {true, false, true};
        assertTrue(Arrays.equals(a, 0, 3, b, 0, 3));
        assertTrue(Arrays.equals(a, 0, 3, a, 0, 3));
        assertFalse(Arrays.equals(a, 0, 2, b, 1, 3));
        boolean[] c = {true, true, true};
        assertFalse(Arrays.equals(a, 0, 3, c, 0, 3));

        boolean[] large = new boolean[100];
        boolean[] large2 = new boolean[100];
        Arrays.fill(large, true);
        Arrays.fill(large2, true);
        // Test various alignment offsets for vectorized implementations
        for (int i = 0; i < 16; i++) {
            assertTrue(Arrays.equals(large, i, 100, large2, i, 100));
        }
    }

    @Test
    public void testEqualsByteRange() {
        byte[] a = {1, 2, 3};
        byte[] b = {1, 2, 3};
        assertTrue(Arrays.equals(a, 0, 3, b, 0, 3));
        assertTrue(Arrays.equals(a, 0, 3, a, 0, 3));
        assertFalse(Arrays.equals(a, 0, 2, b, 1, 3));
        byte[] c = {1, 9, 3};
        assertFalse(Arrays.equals(a, 0, 3, c, 0, 3));

        byte[] large = new byte[100];
        byte[] large2 = new byte[100];
        Arrays.fill(large, (byte) 1);
        Arrays.fill(large2, (byte) 1);
        // Test various alignment offsets for vectorized implementations
        for (int i = 0; i < 16; i++) {
            assertTrue(Arrays.equals(large, i, 100, large2, i, 100));
        }
    }

    @Test
    public void testEqualsByteArray_nulls() {
        assertTrue(Arrays.equals((byte[]) null, (byte[]) null));
        assertFalse(Arrays.equals(new byte[0], null));
        assertFalse(Arrays.equals(null, new byte[0]));
    }

    @Test
    public void testEqualsByteArray_empty() {
        assertTrue(Arrays.equals(new byte[]{}, new byte[]{}));
    }

    @Test
    public void testEqualsByteArray_sameInstance() {
        byte[] a = byteTestArray(32, 42);
        assertTrue(Arrays.equals(a, a));
    }

    @Test
    public void testEqualsByteArray_equalContent() {
        // Test various sizes to exercise vectorized and tail loops
        assertTrue(Arrays.equals(byteTestArray(1, 42), byteTestArray(1, 42)));
        assertTrue(Arrays.equals(byteTestArray(7, 42), byteTestArray(7, 42)));
        assertTrue(Arrays.equals(byteTestArray(16, 42), byteTestArray(16, 42)));
        assertTrue(Arrays.equals(byteTestArray(33, 42), byteTestArray(33, 42)));
        assertTrue(Arrays.equals(byteTestArray(100, 42), byteTestArray(100, 42)));
    }

    @Test
    public void testEqualsByteArray_differentLengths() {
        byte[] a = byteTestArray(32, 42);
        byte[] b = Arrays.copyOf(a, a.length + 1);
        byte[] c = Arrays.copyOf(a, a.length - 1);

        assertFalse(Arrays.equals(a, b));
        assertFalse(Arrays.equals(a, c));
        assertFalse(Arrays.equals(new byte[0], new byte[1]));
    }

    @Test
    public void testEqualsByteArray_mismatchAtStart() {
        byte[] a = byteTestArray(32, 42);
        byte[] b = a.clone();
        b[0] = (byte) (a[0] + 1);
        assertFalse(Arrays.equals(a, b));
    }

    @Test
    public void testEqualsByteArray_mismatchInMiddle() {
        byte[] a = byteTestArray(32, 42);
        byte[] b = a.clone();
        b[15] = (byte) (a[15] + 1);
        assertFalse(Arrays.equals(a, b));
    }

    @Test
    public void testEqualsByteArray_mismatchAtEnd() {
        byte[] a = byteTestArray(32, 42);
        byte[] b = a.clone();
        b[31] = (byte) (a[31] + 1);
        assertFalse(Arrays.equals(a, b));
    }

    @Test
    public void testEqualsCharRange() {
        char[] a = {'a', 'b', 'c'};
        char[] b = {'a', 'b', 'c'};
        assertTrue(Arrays.equals(a, 0, 3, b, 0, 3));
        assertTrue(Arrays.equals(a, 0, 3, a, 0, 3));
        assertFalse(Arrays.equals(a, 0, 2, b, 1, 3));
        char[] c = {'a', 'd', 'c'};
        assertFalse(Arrays.equals(a, 0, 3, c, 0, 3));

        char[] large = new char[100];
        char[] large2 = new char[100];
        Arrays.fill(large, 'a');
        Arrays.fill(large2, 'a');
        // Test various alignment offsets for vectorized implementations
        for (int i = 0; i < 16; i++) {
            assertTrue(Arrays.equals(large, i, 100, large2, i, 100));
        }
    }

    @Test
    public void testEqualsShortRange() {
        short[] a = {1, 2, 3};
        short[] b = {1, 2, 3};
        assertTrue(Arrays.equals(a, 0, 3, b, 0, 3));
        assertTrue(Arrays.equals(a, 0, 3, a, 0, 3));
        assertFalse(Arrays.equals(a, 0, 2, b, 1, 3));
        short[] c = {1, 9, 3};
        assertFalse(Arrays.equals(a, 0, 3, c, 0, 3));

        short[] large = new short[100];
        short[] large2 = new short[100];
        Arrays.fill(large, (short) 1);
        Arrays.fill(large2, (short) 1);
        // Test various alignment offsets for vectorized implementations
        for (int i = 0; i < 16; i++) {
            assertTrue(Arrays.equals(large, i, 100, large2, i, 100));
        }
    }

    @Test
    public void testEqualsIntRange() {
        int[] a = {1, 2, 3};
        int[] b = {1, 2, 3};
        assertTrue(Arrays.equals(a, 0, 3, b, 0, 3));
        assertTrue(Arrays.equals(a, 0, 3, a, 0, 3));
        assertFalse(Arrays.equals(a, 0, 2, b, 1, 3));
        int[] c = {1, 9, 3};
        assertFalse(Arrays.equals(a, 0, 3, c, 0, 3));

        int[] large = new int[100];
        int[] large2 = new int[100];
        Arrays.fill(large, 1);
        Arrays.fill(large2, 1);
        // Test various alignment offsets for vectorized implementations
        for (int i = 0; i < 16; i++) {
            assertTrue(Arrays.equals(large, i, 100, large2, i, 100));
        }
    }

    @Test
    public void testEqualsByte() {
        // Test zero-length arrays
        assertTrue(Arrays.equals(new byte[0], new byte[0]));
        // Test various lengths to cover vectorized loop and tail loop
        for (int len : IntStream.rangeClosed(1, 65).toArray()) {
            byte[] a = new byte[len];
            byte[] b = new byte[len];
            Arrays.fill(a, (byte) 1);
            Arrays.fill(b, (byte) 1);
            assertTrue("Failed equals for length " + len, Arrays.equals(a, b));
            for (int breakPos = 0; breakPos < len; ++breakPos) {

                b[breakPos] = (byte) 2;
                String msg = "Failed mismatch when elements at " + breakPos +
                        " differ for length " + len;
                assertFalse(msg, Arrays.equals(a, b));
                b[breakPos] = (byte) 1;
            }
        }
    }

    @Test
    public void testEqualsLongRange() {
        long[] a = {1, 2, 3};
        long[] b = {1, 2, 3};
        assertTrue(Arrays.equals(a, 0, 3, b, 0, 3));
        assertTrue(Arrays.equals(a, 0, 3, a, 0, 3));
        assertFalse(Arrays.equals(a, 0, 2, b, 1, 3));
        long[] c = {1, 9, 3};
        assertFalse(Arrays.equals(a, 0, 3, c, 0, 3));

        long[] large = new long[100];
        long[] large2 = new long[100];
        Arrays.fill(large, 1);
        Arrays.fill(large2, 1);
        // Test various alignment offsets for vectorized implementations
        for (int i = 0; i < 16; i++) {
            assertTrue(Arrays.equals(large, i, 100, large2, i, 100));
        }
    }

    @Test
    public void testEqualsIntTailHandling() {
        // Test array lengths 0 to 32 to cover various tail scenarios and small arrays
        for (int len = 0; len <= 32; len++) {
            int[] a = new int[len];
            int[] b = new int[len];
            Arrays.fill(a, 1);
            Arrays.fill(b, 1);
            assertTrue("Length " + len, Arrays.equals(a, b));
            assertTrue("Range Length " + len, Arrays.equals(a, 0, len, b, 0, len));

            if (len == 0) {
                continue;
            }

            b[0] = 2;
            assertFalse("Mismatch 0 length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch 0 length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[0] = 1;

            b[len - 1] = 2;
            assertFalse("Mismatch end length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch end length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[len - 1] = 1;
        }
    }

    @Test
    public void testEqualsByteTailHandling() {
        for (int len = 0; len <= 32; len++) {
            byte[] a = new byte[len];
            byte[] b = new byte[len];
            Arrays.fill(a, (byte) 1);
            Arrays.fill(b, (byte) 1);
            assertTrue("Length " + len, Arrays.equals(a, b));
            assertTrue("Range Length " + len, Arrays.equals(a, 0, len, b, 0, len));

            if (len == 0) {
                continue;
            }

            b[0] = (byte) 2;
            assertFalse("Mismatch 0 length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch 0 length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[0] = (byte) 1;

            b[len - 1] = (byte) 2;
            assertFalse("Mismatch end length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch end length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[len - 1] = (byte) 1;
        }
    }

    @Test
    public void testEqualsShortTailHandling() {
        for (int len = 0; len <= 32; len++) {
            short[] a = new short[len];
            short[] b = new short[len];
            Arrays.fill(a, (short) 1);
            Arrays.fill(b, (short) 1);
            assertTrue("Length " + len, Arrays.equals(a, b));
            assertTrue("Range Length " + len, Arrays.equals(a, 0, len, b, 0, len));

            if (len == 0) {
                continue;
            }

            b[0] = (short) 2;
            assertFalse("Mismatch 0 length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch 0 length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[0] = (short) 1;

            b[len - 1] = (short) 2;
            assertFalse("Mismatch end length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch end length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[len - 1] = (short) 1;
        }
    }

    @Test
    public void testEqualsCharTailHandling() {
        for (int len = 0; len <= 32; len++) {
            char[] a = new char[len];
            char[] b = new char[len];
            Arrays.fill(a, 'a');
            Arrays.fill(b, 'a');
            assertTrue("Length " + len, Arrays.equals(a, b));
            assertTrue("Range Length " + len, Arrays.equals(a, 0, len, b, 0, len));

            if (len == 0) {
                continue;
            }

            b[0] = 'b';
            assertFalse("Mismatch 0 length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch 0 length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[0] = 'a';

            b[len - 1] = 'b';
            assertFalse("Mismatch end length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch end length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[len - 1] = 'a';
        }
    }

    @Test
    public void testEqualsLongTailHandling() {
        for (int len = 0; len <= 32; len++) {
            long[] a = new long[len];
            long[] b = new long[len];
            Arrays.fill(a, 1L);
            Arrays.fill(b, 1L);
            assertTrue("Length " + len, Arrays.equals(a, b));
            assertTrue("Range Length " + len, Arrays.equals(a, 0, len, b, 0, len));

            if (len == 0) {
                continue;
            }

            b[0] = 2L;
            assertFalse("Mismatch 0 length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch 0 length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[0] = 1L;

            b[len - 1] = 2L;
            assertFalse("Mismatch end length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch end length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[len - 1] = 1L;
        }
    }

    @Test
    public void testEqualsBooleanTailHandling() {
        for (int len = 0; len <= 32; len++) {
            boolean[] a = new boolean[len];
            boolean[] b = new boolean[len];
            Arrays.fill(a, true);
            Arrays.fill(b, true);
            assertTrue("Length " + len, Arrays.equals(a, b));
            assertTrue("Range Length " + len, Arrays.equals(a, 0, len, b, 0, len));

            if (len == 0) {
                continue;
            }

            b[0] = false;
            assertFalse("Mismatch 0 length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch 0 length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[0] = true;

            b[len - 1] = false;
            assertFalse("Mismatch end length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch end length " + len, Arrays.equals(a, 0, len, b, 0, len));
            b[len - 1] = true;
        }
    }

    @Test
    public void testEqualsFloatTailHandling() {
        for (int len = 0; len <= 32; len++) {
            float[] a = new float[len];
            float[] b = new float[len];
            Arrays.fill(a, 1.0f);
            Arrays.fill(b, 1.0f);
            assertTrue("Length " + len, Arrays.equals(a, b));
            assertTrue("Range Length " + len, Arrays.equals(a, 0, len, b, 0, len));

            if (len == 0) {
                continue;
            }

            b[len - 1] = 2.0f;
            assertFalse("Mismatch end length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch end length " + len,
                    Arrays.equals(a, 0, len, b, 0, len));
            b[len - 1] = 1.0f;

            // Signed zero
            a[0] = 0.0f;
            b[0] = -0.0f;
            assertFalse("0.0 equals -0.0 length " + len, Arrays.equals(a, b));
            assertFalse("Range 0.0 equals -0.0 length " + len,
                    Arrays.equals(a, 0, len, b, 0, len));

            // NaN checks
            a[0] = Float.NaN;
            b[0] = Float.NaN;
            assertTrue("NaN equals NaN length " + len, Arrays.equals(a, b));
            assertTrue("Range NaN equals NaN length " + len,
                    Arrays.equals(a, 0, len, b, 0, len));

            // Different NaN bit patterns
            a[0] = Float.intBitsToFloat(0x7fc00000); // Standard NaN
            b[0] = Float.intBitsToFloat(0x7fc00001); // Different NaN
            assertTrue("NaN(1) equals NaN(2) length " + len, Arrays.equals(a, b));
            assertTrue("Range NaN(1) equals NaN(2) length " + len,
                    Arrays.equals(a, 0, len, b, 0, len));

            // Mixed NaN and non-NaN
            a[0] = 1.0f;
            b[0] = Float.NaN;
            assertFalse("1.0 equals NaN length " + len, Arrays.equals(a, b));
            assertFalse("Range 1.0 equals NaN length " + len,
                    Arrays.equals(a, 0, len, b, 0, len));
        }
    }

    @Test
    public void testEqualsDoubleTailHandling() {
        for (int len = 0; len <= 32; len++) {
            double[] a = new double[len];
            double[] b = new double[len];
            Arrays.fill(a, 1.0d);
            Arrays.fill(b, 1.0d);
            assertTrue("Length " + len, Arrays.equals(a, b));
            assertTrue("Range Length " + len, Arrays.equals(a, 0, len, b, 0, len));

            if (len == 0) {
                continue;
            }

            b[len - 1] = 2.0d;
            assertFalse("Mismatch end length " + len, Arrays.equals(a, b));
            assertFalse("Range Mismatch end length " + len,
                    Arrays.equals(a, 0, len, b, 0, len));
            b[len - 1] = 1.0d;

            // Signed zero
            a[0] = 0.0d;
            b[0] = -0.0d;
            assertFalse("0.0 equals -0.0 length " + len, Arrays.equals(a, b));
            assertFalse("Range 0.0 equals -0.0 length " + len,
                    Arrays.equals(a, 0, len, b, 0, len));

            // NaN checks
            a[0] = Double.NaN;
            b[0] = Double.NaN;
            assertTrue("NaN equals NaN length " + len, Arrays.equals(a, b));
            assertTrue("Range NaN equals NaN length " + len,
                    Arrays.equals(a, 0, len, b, 0, len));

            // Different NaN bit patterns
            a[0] = Double.longBitsToDouble(0x7ff8000000000000L); // Standard NaN
            b[0] = Double.longBitsToDouble(0x7ff8000000000001L); // Different NaN
            assertTrue("NaN(1) equals NaN(2) length " + len, Arrays.equals(a, b));
            assertTrue("Range NaN(1) equals NaN(2) length " + len,
                    Arrays.equals(a, 0, len, b, 0, len));

            // Mixed NaN and non-NaN
            a[0] = 1.0d;
            b[0] = Double.NaN;
            assertFalse("1.0 equals NaN length " + len, Arrays.equals(a, b));
            assertFalse("Range 1.0 equals NaN length " + len,
                    Arrays.equals(a, 0, len, b, 0, len));
        }
    }
}
