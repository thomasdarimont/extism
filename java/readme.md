Java Example
---

The following example uses the experimental [Panama](https://openjdk.org/projects/panama/) [support available for Java 19](https://jdk.java.net/panama/).

# Setup

## JDK 19 with Panama Support

This example needs a Java 19 version with Panama support.

A simple way to install the proper JDK is to use [SDKMan](https://sdkman.io/):
```
sdk install java 19.ea.1.pma-open
```

```
$ java -version                    
openjdk version "19-panama" 2022-09-20
OpenJDK Runtime Environment (build 19-panama+1-13)
OpenJDK 64-Bit Server VM (build 19-panama+1-13, mixed mode, sharing)
```

## Build libextism

Ensure that `libexitsm.so` exsists in `../target/release`, see the instructions in [readme.md](../README.md)

# Generate Java bindings with jextract

```
jextract ../runtime/extism.h --source -t com.github.extism -d src/main/java
```

# Build the example

```
mvn clean package
```

# Run the example

```
java \
  --add-modules jdk.incubator.foreign  \
  --enable-native-access=ALL-UNNAMED \
  --enable-preview \
  -cp target/classes \
  -Djava.library.path=../target/release/libextism.so \
  -Dinput="AABBCCDDEE" \
  com.github.extism.demo.ExtismDemo
```

Output:
```
WARNING: Using incubator modules: jdk.incubator.foreign
{"count": 4}
```
