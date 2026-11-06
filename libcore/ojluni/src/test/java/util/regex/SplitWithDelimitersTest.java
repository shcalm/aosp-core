/*
 * Copyright (c) 2023, Oracle and/or its affiliates. All rights reserved.
 * DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS FILE HEADER.
 *
 * This code is free software; you can redistribute it and/or modify it
 * under the terms of the GNU General Public License version 2 only, as
 * published by the Free Software Foundation.
 *
 * This code is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
 * FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License
 * version 2 for more details (a copy is included in the LICENSE file that
 * accompanied this code).
 *
 * You should have received a copy of the GNU General Public License version
 * 2 along with this work; if not, write to the Free Software Foundation,
 * Inc., 51 Franklin St, Fifth Floor, Boston, MA 02110-1301 USA.
 *
 * Please contact Oracle, 500 Oracle Parkway, Redwood Shores, CA 94065 USA
 * or visit www.oracle.com if you need additional information or have any
 * questions.
 */

/*
 * @test
 * @bug 8305486
 * @summary Tests to exercise the split functionality added in the issue.
 * @run junit SplitWithDelimitersTest
 */

package test.java.util.regex;

import static org.junit.Assert.assertArrayEquals;

import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.Parameterized;
import org.junit.runners.Parameterized.Parameters;

import java.util.Arrays;
import java.util.Collection;
import java.util.regex.Pattern;

// Android-changed: Rewritten in JUnit 4 style to avoid dependency on the JUnit Jupiter.
@RunWith(Parameterized.class)
public class SplitWithDelimitersTest {

    private final String[] expected;
    private final String target;
    private final String regex;
    private final int limit;

    public SplitWithDelimitersTest(String[] expected, String target, String regex, int limit) {
        this.expected = expected;
        this.target = target;
        this.regex = regex;
        this.limit = limit;
    }

    private static String[] dropOddIndexed(String[] a, int limit) {
        String[] r = new String[(a.length + 1) / 2];
        for (int i = 0; i < a.length; i += 2) {
            r[i / 2] = a[i];
        }
        int len = r.length;
        if (limit == 0) {
            /* Also drop trailing empty strings */
            for (; len > 0 && r[len - 1].isEmpty(); --len);  // empty body
        }
        return len < r.length ? Arrays.copyOf(r, len) : r;
    }

    @Parameters
    public static Collection<Object[]> data() {
        return Arrays.asList(new Object[][] {
                {new String[] {"b", "o", "", "o", ":::and::f", "o", "", "o", ""},
                        "boo:::and::foo", "o", 5},
                {new String[] {"b", "o", "", "o", ":::and::f", "o", "o"},
                        "boo:::and::foo", "o", 4},
                {new String[] {"b", "o", "", "o", ":::and::foo"},
                        "boo:::and::foo", "o", 3},
                {new String[] {"b", "o", "o:::and::foo"},
                        "boo:::and::foo", "o", 2},
                {new String[] {"boo:::and::foo"},
                        "boo:::and::foo", "o", 1},
                {new String[] {"b", "o", "", "o", ":::and::f", "o", "", "o"},
                        "boo:::and::foo", "o", 0},
                {new String[] {"b", "o", "", "o", ":::and::f", "o", "", "o", ""},
                        "boo:::and::foo", "o", -1},

                {new String[] {"boo", ":::", "and", "::", "foo"},
                        "boo:::and::foo", ":+", 3},
                {new String[] {"boo", ":::", "and::foo"},
                        "boo:::and::foo", ":+", 2},
                {new String[] {"boo:::and::foo"},
                        "boo:::and::foo", ":+", 1},
                {new String[] {"boo", ":::", "and", "::", "foo"},
                        "boo:::and::foo", ":+", 0},
                {new String[] {"boo", ":::", "and", "::", "foo"},
                        "boo:::and::foo", ":+", -1},

                {new String[] {"b", "", "b", "", ""},
                        "bb", "a*|b*", 3},
                {new String[] {"b", "", "b"},
                        "bb", "a*|b*", 2},
                {new String[] {"bb"},
                        "bb", "a*|b*", 1},
                {new String[] {"b", "", "b"},
                        "bb", "a*|b*", 0},
                {new String[] {"b", "", "b", "", ""},
                        "bb", "a*|b*", -1},

                {new String[] {"", "bb", "", "", ""},
                        "bb", "b*|a*", 3},
                {new String[] {"", "bb", ""},
                        "bb", "b*|a*", 2},
                {new String[] {"bb"},
                        "bb", "b*|a*", 1},
                {new String[] {"", "bb"},
                        "bb", "b*|a*", 0},
                {new String[] {"", "bb", "", "", ""},
                        "bb", "b*|a*", -1},
        });
    }

    @Test
    public void testSplit() {
        String[] computedWith = target.splitWithDelimiters(regex, limit);
        assertArrayEquals(expected, computedWith);
        String[] patComputedWith = Pattern.compile(regex).splitWithDelimiters(target, limit);
        assertArrayEquals(computedWith, patComputedWith);

        String[] computedWithout = target.split(regex, limit);
        assertArrayEquals(dropOddIndexed(expected, limit), computedWithout);
        String[] patComputedWithout = Pattern.compile(regex).split(target, limit);
        assertArrayEquals(computedWithout, patComputedWithout);
    }

}