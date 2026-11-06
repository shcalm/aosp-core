-keep interface android.annotation.SystemApi
-keep @android.annotation.SystemApi public class * {
    public protected *;
}
-keepclasseswithmembers public class * {
    @android.annotation.SystemApi public protected *;
}
# Also ensure nested classes are kept. This is overly conservative, but handles
# cases where such classes aren't explicitly marked @SystemApi.
# TODO(b/248580093): Rely on Metalava-generated Proguard rules instead.
-if @android.annotation.SystemApi class *
-keep public class <1>$** {
    public protected *;
}
