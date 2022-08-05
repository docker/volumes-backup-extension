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

export default function LoadDialog({...props}) {
    console.log("LoadDialog component rendered.");
    const ddClient = useDockerDesktopClient();

    const context = useContext(MyContext);

    const [imageName, setImageName] = React.useState<string>("");
    const [actionInProgress, setActionInProgress] =
        React.useState<boolean>(false);

    const loadImage = async () => {
        setActionInProgress(true);
        let actionSuccessfullyCompleted = false

        ddClient.extension.vm.service
            .get(`/volumes/${context.store.volume.volumeName}/load?image=${imageName}`)
            .then((_: any) => {
                actionSuccessfullyCompleted = true
                ddClient.desktopUI.toast.success(
                    `Copied /volume-data from image ${imageName} into volume ${context.store.volume.volumeName}`
                );
            })
            .catch((error) => {
                console.log(error)
                actionSuccessfullyCompleted = false
                ddClient.desktopUI.toast.error(
                    `Failed to copy /volume-data from image ${imageName} to into volume ${context.store.volume.volumeName}: ${error.message}. HTTP status code: ${error.statusCode}`
                );
            })
            .finally(() => {
                setActionInProgress(false);
                setImageName("");
                props.onClose(actionSuccessfullyCompleted)
            })
    };

    return (
        <Dialog open={props.open} onClose={props.onClose}>
            <DialogTitle>Load a directory from an image to a volume</DialogTitle>
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
                    Copies /volume-data contents from an image to a volume
                </DialogContentText>

                <Grid container direction="column" spacing={2}>
                    <Grid item>
                        <TextField
                            required
                            autoFocus
                            margin="dense"
                            id="image-name"
                            label="Image name"
                            fullWidth
                            variant="standard"
                            placeholder={`my-image:latest`}
                            spellCheck={false}
                            onChange={(e) => {
                                setImageName(e.target.value);
                            }}
                        />
                    </Grid>

                    {imageName !== "" && (
                        <Grid item>
                            <Typography variant="body1" color="text.secondary" sx={{mb: 2}}>
                                The /volume-data from image {imageName} will be copied to volume{" "}
                                {context.store.volume.volumeName}.
                            </Typography>
                            <Typography variant="body1" color="text.secondary" sx={{mb: 2}}>
                                Once the operation is completed, you can inspect the volume
                                contents of {context.store.volume.volumeName} or export its content
                                into a local directory.
                            </Typography>
                            <Typography variant="body1" color="text.secondary">
                                ⚠️ This will replace any existing data inside the volume.
                            </Typography>
                        </Grid>
                    )}
                </Grid>
            </DialogContent>
            <DialogActions>
                <Button
                    onClick={() => {
                        setImageName("");
                        props.onClose(false);
                    }}
                >
                    Cancel
                </Button>
                <Button onClick={loadImage} disabled={imageName === ""}>
                    Load
                </Button>
            </DialogActions>
        </Dialog>
    );
}
