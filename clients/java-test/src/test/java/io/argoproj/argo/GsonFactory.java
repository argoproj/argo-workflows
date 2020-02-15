package io.argoproj.argo;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonDeserializer;
import com.google.gson.JsonParseException;
import io.argoproj.argo.model.V1Time;

import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.Date;

public class GsonFactory {

    /*
       We need a special deserializer for dates.
     */
    static final Gson GSON = new GsonBuilder().registerTypeAdapter(V1Time.class, (JsonDeserializer<V1Time>) (json, typeOfT, context) -> {
        try {
            Date date = new SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'").parse(json.getAsString());
            return new V1Time().nanos(1000 * (int) (date.getTime()));
        } catch (ParseException e) {
            throw new JsonParseException(e);
        }
    }).create();

}
