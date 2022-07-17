import React, {useContext} from "react";
import {Backdrop, Button, CircularProgress, Grid, Typography,} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import {createDockerDesktopClient} from "@docker/extension-api-client";

import {MyContext} from "../index";
import {isError} from "../common/isError";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
    return client;
}

export default function ImportDialog({...props}) {
    console.log("ImportDialog component rendered.");
    const ddClient = useDockerDesktopClient();

    const context = useContext(MyContext);

    const [path, setPath] = React.useState<string>("");
    const [actionInProgress, setActionInProgress] =
        React.useState<boolean>(false);

    const selectImportTarGzFile = () => {
        ddClient.desktopUI.dialog
            .showOpenDialog({
                properties: ["openFile"],
                filters: [{name: ".tar.gz", extensions: ["tar.gz"]}], // should contain extension without wildcards or dots
            })
            .then((result) => {
                if (result.canceled) {
                    return;
                }

                setPath(result.filePaths[0]);
            });
    };

    const importVolume = async () => {
        setActionInProgress(true);
        let actionSuccessfullyCompleted = false

        ddClient.extension.vm.service
            .get(`/volumes/${context.store.volumeName}/import?path=${path}`)
            .then((_: any) => {
                actionSuccessfullyCompleted = true
                ddClient.desktopUI.toast.success(
                    `File ${path} imported into volume ${context.store.volumeName}`
                );
            })
            .catch((error) => {
                actionSuccessfullyCompleted = false
                ddClient.desktopUI.toast.error(
                    `Failed to import file ${path} into volume ${context.store.volumeName}: ${error.stderr} Exit code: ${error.code}`
                );
            })
            .finally(() => {
                setActionInProgress(false);
                setPath("");
                props.onClose(actionSuccessfullyCompleted)
            })
    };

    return (
        <Dialog open={props.open} onClose={props.onClose}>
            <DialogTitle>Import gzip'ed tarball into a volume</DialogTitle>
            <DialogContent>
                <Backdrop
                    sx={{
                        backgroundColor: "rgba(245,244,244,0.4)",
                        zIndex: (theme) => theme.zIndex.drawer + 1,
                    }}
                    open={actionInProgress}
                >
                    <CircularProgress color="info"/>
                </Backdrop>
                <DialogContentText>
                    Extracts a gzip'ed tarball into a volume.
                </DialogContentText>

                <Grid container direction="column" spacing={2}>
                    <Grid item>
                        <Typography variant="body1" color="text.secondary" sx={{mt: 2}}>
                            Choose a .tar.gz to import, e.g. file.tar.gz
                        </Typography>
                    </Grid>
                    <Grid item>
                        <Button variant="contained" onClick={selectImportTarGzFile}>
                            Select file
                        </Button>
                    </Grid>

                    {path !== "" && (
                        <Grid item>
                            <Typography variant="body1" color="text.secondary" sx={{mb: 2}}>
                                The file {path} will be imported into volume{" "}
                                {context.store.volumeName}.
                            </Typography>
                            <Typography variant="body1" color="text.secondary">
                                ⚠️ This will replace all the existing data inside the volume.
                            </Typography>
                        </Grid>
                    )}
                </Grid>
            </DialogContent>
            <DialogActions>
                <Button
                    onClick={() => {
                        setPath("");
                        props.onClose(false)
                    }}
                >
                    Cancel
                </Button>
                <Button
                    onClick={importVolume}
                    disabled={path === ""}
                >
                    Import
                </Button>
            </DialogActions>
        </Dialog>
    );
}
