lazy val root = (project in file(".")).
  settings(
    organization := "io.argoproj.argo",
    name := "argo-workflows-java-client",
    version := "latest",
    scalaVersion := "2.11.4",
    scalacOptions ++= Seq("-feature"),
    javacOptions in compile ++= Seq("-Xlint:deprecation"),
    publishArtifact in (Compile, packageDoc) := false,
    resolvers += Resolver.mavenLocal,
    libraryDependencies ++= Seq(
      "io.swagger" % "swagger-annotations" % "1.5.22",
      "com.squareup.okhttp3" % "okhttp" % "3.14.2",
      "com.squareup.okhttp3" % "logging-interceptor" % "3.14.2",
      "com.google.code.gson" % "gson" % "2.8.5",
      "org.apache.commons" % "commons-lang3" % "3.9",
      "org.threeten" % "threetenbp" % "1.4.0" % "compile",
      "io.gsonfire" % "gson-fire" % "1.8.3" % "compile",
      "javax.annotation" % "jsr250-api" % "1.0" % "compile",
      "junit" % "junit" % "4.13" % "test",
      "com.novocode" % "junit-interface" % "0.10" % "test"
    )
  )
