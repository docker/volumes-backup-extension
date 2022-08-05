import React, {useContext} from "react";
import {Backdrop, Button, CircularProgress, Grid, TextField, Typography,} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import {createDockerDesktopClient} from "@docker/extension-api-client";

import {MyContext} from "../index";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
    return client;
}

export default function ExportDialog({...props}) {
    console.log("ExportDialog component rendered.");
    const ddClient = useDockerDesktopClient();
    const context = useContext(MyContext);

    const [fileName, setFileName] = React.useState<string>(
        `${context.store.volume.volumeName}.tar.gz`
    );
    const [path, setPath] = React.useState<string>("");
    const [actionInProgress, setActionInProgress] =
        React.useState<boolean>(false);

    const selectExportDirectory = () => {
        ddClient.desktopUI.dialog
            .showOpenDialog({
                properties: ["openDirectory"],
            })
            .then((result) => {
                if (result.canceled) {
                    return;
                }

                setPath(result.filePaths[0]);
            });
    };

    const exportVolume = async () => {
        setActionInProgress(true);

        console.log("volume name:", context.store.volume.volumeName);
        console.log("path:", path);
        console.log("fileName:", fileName);

        // const encodedPath = encodeURIComponent(path)
        // console.log("encodedPath:", encodedPath);

        ddClient.extension.vm.service
            .get(`/volumes/${context.store.volume.volumeName}/export?path=${path}&fileName=${fileName}`)
            .then((_: any) => {
                ddClient.desktopUI.toast.success(
                    `Volume ${context.store.volume.volumeName} exported to ${path}`
                );
            })
            .catch((error) => {
                ddClient.desktopUI.toast.error(
                    `Failed to backup volume ${context.store.volume.volumeName} to ${path}: ${error.message}. HTTP status code: ${error.statusCode}`
                );
            })
            .finally(() => {
                setActionInProgress(false);
                props.onClose();
            })
    };

    return (
        <Dialog open={props.open} onClose={props.onClose}>
            <DialogTitle>Export volume to local directory</DialogTitle>
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
                    Creates a gzip'ed tarball in the selected directory from a volume.
                </DialogContentText>

                <Grid container direction="column" spacing={2}>
                    <Grid item>
                        <TextField
                            required
                            autoFocus
                            margin="dense"
                            id="file-name"
                            label="File name"
                            fullWidth
                            variant="standard"
                            defaultValue={`${context.store.volume.volumeName}.tar.gz`}
                            spellCheck={false}
                            onChange={(e) => {
                                setFileName(e.target.value);
                            }}
                        />
                    </Grid>
                    <Grid item>
                        <Button variant="contained" onClick={selectExportDirectory}>
                            Select directory
                        </Button>
                    </Grid>

                    {path !== "" && (
                        <Grid item>
                            <Typography variant="body1" color="text.secondary">
                                The volume will be exported to {path}/{fileName}
                            </Typography>
                        </Grid>
                    )}
                </Grid>
            </DialogContent>
            <DialogActions>
                <Button onClick={props.onClose}>Cancel</Button>
                <Button
                    onClick={exportVolume}
                    disabled={path === "" || fileName === ""}
                >
                    Export
                </Button>
            </DialogActions>
        </Dialog>
    );
}
