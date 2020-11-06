import * as React from "react";


export const EventsHelp = (props: { className?: string }) => <div className={props.className || ""}>
    <p>Argo Events allow you to trigger workflows, lambadas, and other actions based on webhooks, message, or a cron
        schedule.</p>
    <p><a href='https://argoproj.github.io/argo-events/'>Learn more</a></p>
</div>